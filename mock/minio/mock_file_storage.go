package mockminio

import "github.com/stretchr/testify/mock"

type MockFileStorage struct {
	mock.Mock
}

func (m *MockFileStorage) Upload(name string, data []byte) (string, error) {
	args := m.Called(name, data)
	return args.String(0), args.Error(1)
}

func (m *MockFileStorage) Delete(objectID string) error {
	args := m.Called(objectID)
	return args.Error(0)
}
