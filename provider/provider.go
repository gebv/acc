package provider

type Provider string

func (p Provider) Match(in Provider) bool {
	return p == in
}

const (
	UNKNOWN_PROVIDER Provider = ""
	INTERNAL         Provider = "internal"
)
