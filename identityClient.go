package main

type identityClient struct {
}

func (ic *identityClient) ByCredentials(user, pass string) (*identity, error) {
	return &identity{}, nil
}

type identity struct {
}

func (i *identity) ID() string {
	return "anonymous"
}

func (i *identity) Claims() map[string]interface{} {
	return map[string]interface{}{
		"fn":   "Joe",
		"role": "basic",
	}
}
