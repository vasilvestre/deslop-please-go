package c

type Port interface{ Send() }

type Impl struct{}

func (Impl) Send() {}
