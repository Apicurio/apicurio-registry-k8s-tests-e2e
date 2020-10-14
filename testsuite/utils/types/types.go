package types

//TestContext holds the common information of for a functional test
type TestContext struct {
	ID      string
	Storage string

	RegistryName string
	RegistryHost string
	RegistryPort string

	cleanupFunctions []func()
}

func (ctx *TestContext) RegisterCleanup(cleanup func()) {
	ctx.cleanupFunctions = append(ctx.cleanupFunctions, cleanup)
}

func (ctx *TestContext) ExecuteCleanups() {
	for i := len(ctx.cleanupFunctions) - 1; i >= 0; i-- {
		ctx.cleanupFunctions[i]()
	}
}
