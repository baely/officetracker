package server

type AuthMethod int

const (
	AuthMethodUnknown = AuthMethod(iota)
	AuthMethodNone
	AuthMethodSSO
	AuthMethodSecret
	AuthMethodExcluded
)
