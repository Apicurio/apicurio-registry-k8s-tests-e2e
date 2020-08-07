package types

//TestContext holds the common information of for a functional test
type TestContext struct {
	ID      string
	Storage string

	RegistryHost string
	RegistryPort string
}
