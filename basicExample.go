// This chaincode will manage two accounts X and Y and will transfer units from X to Y upon invocation

package main

import (
  "errors"
  "fmt"
  "strconv"

  "github.com/hyperledger/fabric/core/chaincode/shim"
)

type SimpleChaincode struct {
}

var X, Y string
var XBalance, YBalance, transfer int

// init callback representing the invocation of a chaincode
func (t *SimpleChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
  var err error

  if len(args) != 4 {
    return nil, errors.New("Incorrect number of arguments. Expecting 4")
  }

  // Initialize the chaincode
  X = args[0]
  XBalance, err = strconv.Atoi(args[1])
  if err != nil {
    return nil, errors.New("Expecting integer value for asset holding")
  }
  Y = args[2]
  YBalance, err = strconv.Atoi(args[3])
  if err != nil {
    return nil, errors.New("Expecting integer value for asset holding")
  }
  ret :=  fmt.Sprintf("Init. Balance in X = %d, balance in Y = %d\n", XBalance, YBalance)
  retbyte := []byte(ret)

  // Write the state to the ledger
  err = stub.PutState(X, []byte(strconv.Itoa(XBalance)))
  if err != nil {
    return nil, err
  }

  err = stub.PutState(Y, []byte(strconv.Itoa(YBalance)))
  if err != nil {
    return nil, err
  }

  return retbyte, nil
}

func (t *SimpleChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {

  // Transaction makes payment of transfer units from X to Y
  var err error

  // get the amount to be transferred
  transfer, err = strconv.Atoi(args[0])

  // get the original values
  XBalanceStr, err := stub.GetState(X)
  XBalance = strconv.Atoi(string(XBalanceStr))
  if err != nil {
  	return nil, errors.New("Error: Failed to get state for X")
  }

  YBalanceStr, err := stub.GetState(Y)
  YBalance = strconv.Atoi(string(YBalanceStr))
  if err != nil {
	return nil, errors.New("Error: Failed to get state for Y")
  }

  // do the transfer
  XBalance = XBalance - transfer
  YBalance = YBalance + transfer

  // write back the updated values to the ledger
  err = stub.PutState(X, []byte(strconv.Itoa(XBalance)))
  if err != nil {
    return nil, err
  }
  err = stub.PutState(Y, []byte(strconv.Itoa(YBalance)))
  if err != nil {
    return nil, err
  }

  ret := fmt.Sprintf("Invoke. Balance in X = %d, balance in Y = %d\n", XBalance, YBalance)
  retbyte := []byte(ret)

  return retbyte, err
}

// Query callback representing the query of a chaincode
func (t *SimpleChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
  
	XBalanceStr, err := stub.GetState(X)
	XBalance = strconv.Atoi(string(XBalanceStr))
	if err != nil {
		return nil, errors.New("Error: Failed to get state for X")
	}
  
	YBalanceStr, err := stub.GetState(Y)
	YBalance = strconv.Atoi(string(YBalanceStr))
	if err != nil {
		return nil, errors.New("Error: Failed to get state for Y")
	}

  ret := fmt.Sprintf("Query. Balance in X = %d, balance in Y = %d\n", XBalance, YBalance)
  retbyte := []byte(ret)

  return retbyte, nil
}

func main() {
  err := shim.Start(new(SimpleChaincode))
  if err != nil {
    fmt.Printf("Error starting Simple chaincode: %s", err)
  }
}
