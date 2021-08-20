package blqueue

type Catcher interface {
	Catch(x interface{})
}
