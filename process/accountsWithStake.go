package process

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ElrondNetwork/elrond-accounts-manager/core"
	"github.com/ElrondNetwork/elrond-accounts-manager/data"
	nodeCore "github.com/ElrondNetwork/elrond-go/core"
	"github.com/tidwall/gjson"
)

const (
	pathValidatorsStake = "/network/direct-staked-info"
	pathDelegatorStake  = "/network/delegated-info"
	pathVMValues        = "/vm-values/query"
	getFullWaitingList  = "getFullWaitingList"
	getFullActiveList   = "getFullActiveList"
	lkMexSnapShot       = "getSnapshot"
)

type accountsGetter struct {
	restClient         RestClientHandler
	pubKeyConverter    nodeCore.PubkeyConverter
	authenticationData data.RestApiAuthenticationData

	delegationContractAddress string
	lkMexContractAddress      string
}

// NewAccountsGetter will create a new instance of accountsGetter
func NewAccountsGetter(
	restClient RestClientHandler,
	delegationContractAddress string,
	pubKeyConverter nodeCore.PubkeyConverter,
	authenticationData data.RestApiAuthenticationData,
	lkMexContractAddress string,
) (*accountsGetter, error) {
	return &accountsGetter{
		restClient:                restClient,
		delegationContractAddress: delegationContractAddress,
		pubKeyConverter:           pubKeyConverter,
		authenticationData:        authenticationData,
		lkMexContractAddress:      lkMexContractAddress,
	}, nil
}

// GetLegacyDelegatorsAccounts will fetch all accounts with stake from API
func (ag *accountsGetter) GetLegacyDelegatorsAccounts() (map[string]*data.AccountInfoWithStakeValues, error) {
	defer logExecutionTime(time.Now(), "Fetched accounts from legacy delegation contract")

	activeListAccounts, err := ag.getFullActiveListAccounts()
	if err != nil {
		return nil, err
	}

	fullWaitingListAccounts, err := ag.getFullWaitingListAccounts()
	if err != nil {
		return nil, err
	}

	accountsMap := make(map[string]*data.AccountInfoWithStakeValues)
	for key, value := range activeListAccounts {
		_, found := accountsMap[key]
		if !found {
			accountsMap[key] = &data.AccountInfoWithStakeValues{
				StakeInfo: data.StakeInfo{
					DelegationLegacyActive:    value,
					DelegationLegacyActiveNum: core.ComputeBalanceAsFloat(value),
				},
			}

			continue
		}

		valueStake, valueStakeNum := computeTotalBalance(value, accountsMap[key].DelegationLegacyActive)

		accountsMap[key].DelegationLegacyActive = valueStake
		accountsMap[key].DelegationLegacyActiveNum = valueStakeNum
	}

	for key, value := range fullWaitingListAccounts {
		_, ok := accountsMap[key]
		if !ok {
			accountsMap[key] = &data.AccountInfoWithStakeValues{
				StakeInfo: data.StakeInfo{
					DelegationLegacyWaiting:    value,
					DelegationLegacyWaitingNum: core.ComputeBalanceAsFloat(value),
				},
			}

			continue
		}

		valueWaiting, valueWaitingNum := computeTotalBalance(value, accountsMap[key].DelegationLegacyWaiting)

		accountsMap[key].DelegationLegacyWaiting = valueWaiting
		accountsMap[key].DelegationLegacyWaitingNum = valueWaitingNum
	}

	return accountsMap, nil
}

func (ag *accountsGetter) getFullActiveListAccounts() (map[string]string, error) {
	return ag.getAccountsVMQuery(getFullActiveList, 2)
}

func (ag *accountsGetter) getFullWaitingListAccounts() (map[string]string, error) {
	return ag.getAccountsVMQuery(getFullWaitingList, 3)
}

func (ag *accountsGetter) getAccountsVMQuery(funcName string, stepForLoop int) (map[string]string, error) {
	vmRequest := &data.VmValueRequest{
		Address:    ag.delegationContractAddress,
		FuncName:   funcName,
		CallerAddr: ag.delegationContractAddress,
	}

	responseVmValue := &data.ResponseVmValue{}
	err := ag.restClient.CallPostRestEndPoint(pathVMValues, vmRequest, responseVmValue, core.GetEmptyApiCredentials())
	if err != nil {
		return nil, err
	}
	if responseVmValue.Error != "" {
		return nil, fmt.Errorf("%s", responseVmValue.Error)
	}

	returnedData := responseVmValue.Data.Data.ReturnData
	accountsStake := make(map[string]string, 0)
	for idx := 0; idx < len(returnedData); idx += stepForLoop {
		address := ag.pubKeyConverter.Encode(returnedData[idx])
		stakedBalance := big.NewInt(0).SetBytes(returnedData[idx+1])

		accountsStake[address] = stakedBalance.String()
	}

	return accountsStake, nil
}

// GetValidatorsAccounts will fetch all validators accounts
func (ag *accountsGetter) GetValidatorsAccounts() (map[string]*data.AccountInfoWithStakeValues, error) {
	defer logExecutionTime(time.Now(), "Fetched accounts from validators contract")

	genericApiResponse := &data.GenericAPIResponse{}
	err := ag.restClient.CallGetRestEndPoint(pathValidatorsStake, genericApiResponse, ag.authenticationData)
	if err != nil {
		return nil, err
	}
	if genericApiResponse.Error != "" {
		return nil, fmt.Errorf("%s", genericApiResponse.Error)
	}

	list := gjson.Get(string(genericApiResponse.Data), "list")
	accountsInfo := make([]data.StakedInfo, 0)
	err = json.Unmarshal([]byte(list.String()), &accountsInfo)
	if err != nil {
		return nil, err
	}

	accountsStake := make(map[string]*data.AccountInfoWithStakeValues)
	for _, acct := range accountsInfo {
		accountsStake[acct.Address] = &data.AccountInfoWithStakeValues{
			StakeInfo: data.StakeInfo{
				ValidatorsActive:    acct.Staked,
				ValidatorsActiveNum: core.ComputeBalanceAsFloat(acct.Staked),
				ValidatorTopUp:      acct.TopUp,
				ValidatorTopUpNum:   core.ComputeBalanceAsFloat(acct.TopUp),
			},
		}
	}

	return accountsStake, nil
}

// GetDelegatorsAccounts will fetch all delegators accounts
func (ag *accountsGetter) GetDelegatorsAccounts() (map[string]*data.AccountInfoWithStakeValues, error) {
	defer logExecutionTime(time.Now(), "Fetched accounts from delegation manager contracts")

	genericApiResponse := &data.GenericAPIResponse{}
	err := ag.restClient.CallGetRestEndPoint(pathDelegatorStake, genericApiResponse, ag.authenticationData)
	if err != nil {
		log.Warn("CallGetRestEndPoint", "error", err.Error())
		return nil, err
	}
	if genericApiResponse.Error != "" {
		return nil, fmt.Errorf("cannot get delegators accounts %s", genericApiResponse.Error)
	}

	list := gjson.Get(string(genericApiResponse.Data), "list")
	accountsInfo := make([]data.DelegatorStake, 0)
	err = json.Unmarshal([]byte(list.String()), &accountsInfo)
	if err != nil {
		log.Warn("cannot unmarshal accounts info", "error", err.Error())
		return nil, err
	}

	accountsStake := make(map[string]*data.AccountInfoWithStakeValues)
	for _, acct := range accountsInfo {
		accountsStake[acct.DelegatorAddress] = &data.AccountInfoWithStakeValues{
			StakeInfo: data.StakeInfo{
				Delegation:    acct.Total,
				DelegationNum: core.ComputeBalanceAsFloat(acct.Total),
			},
		}
	}

	return accountsStake, nil
}

// GetLKMEXStakeAccounts will fetch all accounts that have stake lkmex tokens
func (ag *accountsGetter) GetLKMEXStakeAccounts() (map[string]*data.AccountInfoWithStakeValues, error) {
	accountsMap := make(map[string]*data.AccountInfoWithStakeValues)
	if ag.lkMexContractAddress == "" {
		return accountsMap, nil
	}

	defer logExecutionTime(time.Now(), "Fetched accounts from lkmex staking contract")

	vmRequest := &data.VmValueRequest{
		Address:    ag.lkMexContractAddress,
		FuncName:   lkMexSnapShot,
		CallerAddr: ag.lkMexContractAddress,
	}

	responseVmValue := &data.ResponseVmValue{}
	err := ag.restClient.CallPostRestEndPoint(pathVMValues, vmRequest, responseVmValue, core.GetEmptyApiCredentials())
	if err != nil {
		return nil, err
	}
	if responseVmValue.Error != "" {
		return nil, fmt.Errorf("%s", responseVmValue.Error)
	}

	stepForLoop := 2
	returnedData := responseVmValue.Data.Data.ReturnData
	accountsStake := make(map[string]string, 0)
	for idx := 0; idx < len(returnedData); idx += stepForLoop {
		address := ag.pubKeyConverter.Encode(returnedData[idx])
		stakedBalance := big.NewInt(0).SetBytes(returnedData[idx+1])

		accountsStake[address] = stakedBalance.String()
	}

	for key, value := range accountsStake {
		accountsMap[key] = &data.AccountInfoWithStakeValues{
			StakeInfo: data.StakeInfo{
				LKMEXStake:    value,
				LKMEXStakeNum: core.ComputeBalanceAsFloat(value),
			},
		}
	}

	return accountsMap, nil
}
