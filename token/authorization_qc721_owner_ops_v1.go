package token

import "errors"

func (n *QC721Token) TransferAuthorizedV1(
	authorization TokenAuthorizationV1,
	to string,
	tokenID uint64,
) (resultErr error) {
	const (
		model     = TokenAuthorizationModelQC721V1
		operation = TokenOperationTransferV1
	)

	caller := authorization.Claim.Caller
	nonce := authorization.Claim.Nonce
	domain := ""
	stage := tokenAuthorizationStageAuthorizationV1

	if n == nil {
		resultErr = errors.New("QC721 token is nil")

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

	n.mu.Lock()

	domain =
		tokenAuthorizationDomainForQC721LockedV1(n)

	defer func() {
		n.mu.Unlock()

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

	if n.authorizationNonces == nil {
		n.authorizationNonces =
			make(map[string]uint64)
	}

	expected := TokenAuthorizationClaimV1{
		Version:   TokenAuthorizationVersionV1,
		Domain:    domain,
		Operation: TokenOperationTransferV1,
		Caller:    caller,
		From:      caller,
		To:        to,
		TokenID:   tokenID,
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
		n.authorizationNonces,
		caller,
		nonce,
	); err != nil {
		resultErr = err
		return resultErr
	}

	stage = tokenAuthorizationStageOperationV1

	if !validTokenAddress(caller) ||
		!validTokenAddress(to) {
		resultErr = ErrInvalidTokenAddress
		return resultErr
	}

	currentOwner, exists := n.owners[tokenID]
	if !exists {
		resultErr = errors.New("token yok")
		return resultErr
	}

	if currentOwner != caller {
		resultErr = errors.New(
			"transfer yetkisi yok",
		)
		return resultErr
	}

	n.owners[tokenID] = to

	commitTokenAuthorizationNonceV1(
		n.authorizationNonces,
		caller,
		nonce,
	)

	return nil
}

func (n *QC721Token) BurnAuthorizedV1(
	authorization TokenAuthorizationV1,
	tokenID uint64,
) (resultErr error) {
	const (
		model     = TokenAuthorizationModelQC721V1
		operation = TokenOperationBurnV1
	)

	caller := authorization.Claim.Caller
	nonce := authorization.Claim.Nonce
	domain := ""
	stage := tokenAuthorizationStageAuthorizationV1

	if n == nil {
		resultErr = errors.New("QC721 token is nil")

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

	n.mu.Lock()

	domain =
		tokenAuthorizationDomainForQC721LockedV1(n)

	defer func() {
		n.mu.Unlock()

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

	if n.authorizationNonces == nil {
		n.authorizationNonces =
			make(map[string]uint64)
	}

	expected := TokenAuthorizationClaimV1{
		Version:   TokenAuthorizationVersionV1,
		Domain:    domain,
		Operation: TokenOperationBurnV1,
		Caller:    caller,
		From:      caller,
		TokenID:   tokenID,
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
		n.authorizationNonces,
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

	currentOwner, exists := n.owners[tokenID]
	if !exists {
		resultErr = errors.New("token yok")
		return resultErr
	}

	if currentOwner != caller {
		resultErr = errors.New("yakma yetkisi yok")
		return resultErr
	}

	delete(n.owners, tokenID)
	delete(n.tokenURIs, tokenID)

	commitTokenAuthorizationNonceV1(
		n.authorizationNonces,
		caller,
		nonce,
	)

	return nil
}
