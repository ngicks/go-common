package contextkey

type keyTy string

func keyTypeFunc(name string) *keyTy {
	return (*keyTy)(&name)
}
