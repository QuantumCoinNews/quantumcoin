package token

import "errors"

var ErrTokenSupplyUnderflow = errors.New(
	"token: total supply underflow",
)

func (t *QC20Token) TransferAuthorizedV1(
	authorization TokenAuthorizationV1,
	to string,
	amount uint64,
) (resultErr error) {
	const (
		model     = TokenAuthorizationModelQC20V1
		operation = TokenOperationTransferV1
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
		Operation: TokenOperationTransferV1,
		Caller:    caller,
		From:      caller,
		To:        to,
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

	if !validTokenAddress(caller) ||
		!validTokenAddress(to) {
		resultErr = ErrInvalidTokenAddress
		return resultErr
	}

	if t.balances[caller] < amount {
		resultErr = errors.New("yetersiz bakiye")
		return resultErr
	}

	newToBalance, err := checkedAddUint64(
		t.balances[to],
		amount,
	)
	if err != nil {
		resultErr = ErrTokenBalanceOverflow
		return resultErr
	}

	t.balances[caller] -= amount
	t.balances[to] = newToBalance

	commitTokenAuthorizationNonceV1(
		t.authorizationNonces,
		caller,
		nonce,
	)

	return nil
}

func (t *QC20Token) BurnAuthorizedV1(
	authorization TokenAuthorizationV1,
	amount uint64,
) (resultErr error) {
	const (
		model     = TokenAuthorizationModelQC20V1
		operation = TokenOperationBurnV1
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
		Operation: TokenOperationBurnV1,
		Caller:    caller,
		From:      caller,
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

	if !validTokenAddress(caller) {
		resultErr = ErrInvalidTokenAddress
		return resultErr
	}

	if amount == 0 {
		resultErr = ErrTokenAmountZero
		return resultErr
	}

	if t.balances[caller] < amount {
		resultErr = errors.New("yetersiz bakiye")
		return resultErr
	}

	if t.totalSupply < amount {
		resultErr = ErrTokenSupplyUnderflow
		return resultErr
	}

	t.balances[caller] -= amount
	t.totalSupply -= amount

	commitTokenAuthorizationNonceV1(
		t.authorizationNonces,
		caller,
		nonce,
	)

	return nil
}
