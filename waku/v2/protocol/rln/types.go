package rln

/*
# inputs of the membership contract constructor
# TODO may be able to make these constants private and put them inside the waku_rln_relay_utils
const
  MEMBERSHIP_FEE* = 5.u256
*/

// TODO the ETH_CLIENT should be an input to the rln-relay, though hardcoded for now
// the current address is the address of ganache-cli when run locally
const ETH_CLIENT = "ws://localhost:8540/"

type MessageValidationResult int

const (
	MessageValidationResult_Unknown MessageValidationResult = iota
	MessageValidationResult_Valid
	MessageValidationResult_Invalid
	MessageValidationResult_Spam
)
