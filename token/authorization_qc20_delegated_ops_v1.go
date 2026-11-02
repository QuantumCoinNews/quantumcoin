package token

import "errors"

// ApproveAuthorizedV1 sets or revokes an allowance using an owner-signed
// operation. An amount of zero revokes the allowance.
func (t *QC20Token) ApproveAuthorizedV1(
	authorization TokenAuthorizationV1,
	spender string,
	amount uint64,
) (resultErr error) {
	const (
		model     = TokenAuthorizationModelQC20V1
		operation = TokenOperationApproveV1
	)

	caller := authorization.Claim.Caller
	nonce := authorization.Claim.Nonce
	domain := ""
	stage := tokenAuthorizationStageAuthorizationV1

	if t == nil {
		resultErr = errors.New("QC20 token is nil")

		observeTokenAuthorizationV1(
			model,
			operation,
			caller,
			domain,
			nonce,
			stage,
			resultErr,
		)

		return resultErr
	}

	t.mu.Lock()

	domain =
		tokenAuthorizationDomainForQC20LockedV1(t)

	defer func() {
		t.mu.Unlock()

		observeTokenAuthorizationV1(
			model,
			operation,
			caller,
			domain,
			nonce,
			stage,
			resultErr,
		)
	}()

	if t.authorizationNonces == nil {
		t.authorizationNonces =
			make(map[string]uint64)
	}

	expected := TokenAuthorizationClaimV1{
		Version:   TokenAuthorizationVersionV1,
		Domain:    domain,
		Operation: TokenOperationApproveV1,
		Caller:    caller,
		From:      caller,
		Spender:   spender,
		Amount:    amount,
		Nonce:     nonce,
	}

	if err := verifyTokenAuthorizationV1(
		authorization,
		expected,
	); err != nil {
		resultErr = err
		return resultErr
	}

	if err := checkTokenAuthorizationNonceV1(
		t.authorizationNonces,
		caller,
		nonce,
	); err != nil {
		resultErr = err
		return resultErr
	}

	stage = tokenAuthorizationStageOperationV1

	if !validTokenAddress(caller) ||
		!validTokenAddress(spender) {
		resultErr = ErrInvalidTokenAddress
		return resultErr
	}

	if t.allowances == nil {
		t.allowances =
			make(map[string]map[string]uint64)
	}

	if _, exists := t.allowances[caller]; !exists {
		t.allowances[caller] =
			make(map[string]uint64)
	}

	t.allowances[caller][spender] = amount

	commitTokenAuthorizationNonceV1(
		t.authorizationNonces,
		caller,
		nonce,
	)

	return nil
}

// TransferFromAuthorizedV1 lets the verified spender use an allowance.
// The spender is derived from the signed caller and is not accepted as a
// separate unverified argument.
func (t *QC20Token) TransferFromAuthorizedV1(
	authorization TokenAuthorizationV1,
	owner string,
	recipient string,
	amount uint64,
) (resultErr error) {
	const (
		model     = TokenAuthorizationModelQC20V1
		operation = TokenOperationTransferFromV1
	)

	caller := authorization.Claim.Caller
	nonce := authorization.Claim.Nonce
	domain := ""
	stage := tokenAuthorizationStageAuthorizationV1

	if t == nil {
		resultErr = errors.New("QC20 token is nil")

		observeTokenAuthorizationV1(
			model,
			operation,
			caller,
			domain,
			nonce,
			stage,
			resultErr,
		)

		return resultErr
	}

	t.mu.Lock()

	domain =
		tokenAuthorizationDomainForQC20LockedV1(t)

	defer func() {
		t.mu.Unlock()

		observeTokenAuthorizationV1(
			model,
			operation,
			caller,
			domain,
			nonce,
			stage,
			resultErr,
		)
	}()

	if t.authorizationNonces == nil {
		t.authorizationNonces =
			make(map[string]uint64)
	}

	expected := TokenAuthorizationClaimV1{
		Version:   TokenAuthorizationVersionV1,
		Domain:    domain,
		Operation: TokenOperationTransferFromV1,
		Caller:    caller,
		From:      owner,
		To:        recipient,
		Spender:   caller,
		Amount:    amount,
		Nonce:     nonce,
	}

	if err := verifyTokenAuthorizationV1(
		authorization,
		expected,
	); err != nil {
		resultErr = err
		return resultErr
	}

	if err := checkTokenAuthorizationNonceV1(
		t.authorizationNonces,
		caller,
		nonce,
	); err != nil {
		resultErr = err
		return resultErr
	}

	stage = tokenAuthorizationStageOperationV1

	if amount == 0 {
		resultErr = ErrTokenAmountZero
		return resultErr
	}

	if !validTokenAddress(owner) ||
		!validTokenAddress(caller) ||
		!validTokenAddress(recipient) {
		resultErr = ErrInvalidTokenAddress
		return resultErr
	}

	allowed := t.allowances[owner][caller]

	if allowed < amount {
		resultErr = errors.New("izin yetersiz")
		return resultErr
	}

	if t.balances[owner] < amount {
		resultErr = errors.New("bakiye yetersiz")
		return resultErr
	}

	newRecipientBalance, err := checkedAddUint64(
		t.balances[recipient],
		amount,
	)
	if err != nil {
		resultErr = ErrTokenBalanceOverflow
		return resultErr
	}

	t.balances[owner] -= amount
	t.balances[recipient] = newRecipientBalance
	t.allowances[owner][caller] -= amount

	commitTokenAuthorizationNonceV1(
		t.authorizationNonces,
		caller,
		nonce,
	)

	return nil
}
