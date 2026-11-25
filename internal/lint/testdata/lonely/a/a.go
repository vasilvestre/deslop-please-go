package a

type Store interface{ Get() string }

type memory struct{}

func (memory) Get() string { return "" }

type Keeper interface{ Keep() }

type keeper struct{}

func (keeper) Keep() {}

type Wrapped interface {
	error
	Wrap()
}

type wrapped struct{}

func (wrapped) Error() string { return "" }
func (wrapped) Wrap()         {}

type Empty interface{}
