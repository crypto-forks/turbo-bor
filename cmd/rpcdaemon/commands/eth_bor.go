package commands

func (api *APIImpl) Test() (string, error) {
	return "Hello world from eth_bor", nil
}
