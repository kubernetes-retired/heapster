package extpoints

type Noop interface {
	Noop() string
}

type StringTransformer interface {
	Transform(input string) string
}

type NoopFactory func() Noop
