package process

import (
	"bytes"

	"github.com/ElrondNetwork/elrond-accounts-manager/data"
)

// ElasticClientHandler defines what an elastic client should be able do
type ElasticClientHandler interface {
	CloneIndex(index, targetIndex string) (cloned bool, err error)
	PutMapping(targetIndex string, body *bytes.Buffer) error
	WaitYellowStatus() error
	DoBulkRequest(buff *bytes.Buffer, index string) error
	DoMultiGet(ids []string, index string) ([]byte, error)
	DoScrollRequestAllDocuments(index string, body []byte, handlerFunc func(responseBytes []byte) error) error
	UnsetReadOnly(index string) error
	IsInterfaceNil() bool
}

// // RestClientHandler defines what a rest client should be able do
type RestClientHandler interface {
	CallGetRestEndPoint(path string, value interface{}, authenticationData data.RestApiAuthenticationData) error
	CallPostRestEndPoint(path string, data interface{}, response interface{}, authenticationData data.RestApiAuthenticationData) error
}

// AccountsIndexerHandler defines what an accounts indexer should be able do
type AccountsIndexerHandler interface {
	GetAccounts(addresses []string, index string) (map[string]*data.AccountInfoWithStakeValues, error)
	IndexAccounts(accounts map[string]*data.AccountInfoWithStakeValues, index string) error
	IsInterfaceNil() bool
}

// AccountsProcessorHandler defines what an accounts processor should be able do
type AccountsProcessorHandler interface {
	GetAllAccountsWithStake() (map[string]*data.AccountInfoWithStakeValues, []string, error)
	ComputeClonedAccountsIndex() (string, error)
	IsInterfaceNil() bool
}

// AccountsGetterHandler defines what an accounts getter should be able do
type AccountsGetterHandler interface {
	GetLegacyDelegatorsAccounts() (map[string]*data.AccountInfoWithStakeValues, error)
	GetValidatorsAccounts() (map[string]*data.AccountInfoWithStakeValues, error)
	GetDelegatorsAccounts() (map[string]*data.AccountInfoWithStakeValues, error)
}

// Cloner defines what a clone should be able to do
type Cloner interface {
	CloneIndex(index, newIndex string, body *bytes.Buffer) error
	IsInterfaceNil() bool
}

// Reindexer defines what a reindexer should be able to do
type Reindexer interface {
	ReindexAccounts(sourceIndex string, destinationIndex string, restAccounts map[string]*data.AccountInfoWithStakeValues) error
	IsInterfaceNil() bool
}

// DataProcessor defines what a data processor should be able to do
type DataProcessor interface {
	ProcessAccountsData() error
}
