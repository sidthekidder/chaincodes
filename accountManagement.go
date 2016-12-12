// asset management chaincode

package main

import (
  "errors"
  "fmt"
  "strconv"

  "github.com/hyperledger/fabric/core/chaincode/shim"
)


type AssetManagementChaincode struct {
}

// var assetIndexStr = "_assetindex"       //name for the key/value that will store a list of all known assets

// type Asset struct{
//   Name string `json:"name"`
//   User string `json:"user"`
//   CreatedAt string `json:"CreatedAt"`
//   UpdatedAt string `json:"UpdatedAt"`
// }

// type Owner struct{
//   Name string `json:"name"`
//   User string `json:"user"`
// }


// init callback representing the invocation of a chaincode
// func (t *SimpleChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
//   var err error

//   if len(args) != 4 {
//     return nil, errors.New("Incorrect number of arguments. Expecting 4")
//   }

//   // Initialize the chaincode
//   X = args[0]
//   XBalance, err = strconv.Atoi(args[1])
//   if err != nil {
//     return nil, errors.New("Expecting integer value for asset holding")
//   }
//   Y = args[2]
//   YBalance, err = strconv.Atoi(args[3])
//   if err != nil {
//     return nil, errors.New("Expecting integer value for asset holding")
//   }
//   ret :=  fmt.Sprintf("Init. Balance in X = %d, balance in Y = %d\n", XBalance, YBalance)
//   retbyte := []byte(ret)

//   // Write the state to the ledger
//   err = stub.PutState(X, []byte(strconv.Itoa(XBalance)))
//   if err != nil {
//     return nil, err
//   }

//   err = stub.PutState(Y, []byte(strconv.Itoa(YBalance)))
//   if err != nil {
//     return nil, err
//   }

//   return retbyte, nil
// }

// Init method will be called during deployment.
// The deploy transaction metadata is supposed to contain the administrator cert
func (t *AssetManagementChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
  var err error

  if len(args) != 0 {
    return nil, errors.New("Incorrect number of arguments. Expecting 0")
  }

  // Create ownership table
  err := stub.CreateTable("AssetsOwnership", []*shim.ColumnDefinition{
    &shim.ColumnDefinition{Name: "Asset", Type: shim.ColumnDefinition_STRING, Key: true},
    &shim.ColumnDefinition{Name: "Owner", Type: shim.ColumnDefinition_BYTES, Key: false},
  })
  if err != nil {
    return nil, errors.New("Failed creating AssetsOwnership table.")
  }

  // The metadata contains the certificate of the administrator
  adminCert, err := stub.GetCallerMetadata()
  if err != nil {
    return nil, errors.New("Failed getting metadata.")
  }
  if len(adminCert) == 0 {
    return nil, errors.New("Invalid admin certificate. Empty.")
  }

  stub.PutState("admin", adminCert)

  return nil, nil
}


func (t *AssetManagementChaincode) create(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

  if len(args) != 2 {
    return nil, errors.New("Incorrect number of arguments. Expecting 2")
  }

  asset := args[0]
  owner, err := base64.StdEncoding.DecodeString(args[1])
  if err != nil {
    return nil, errors.New("Failed decodinf owner")
  }

  // Verify the identity of the caller
  // Only an administrator can invoker assign
  adminCertificate, err := stub.GetState("admin")
  if err != nil {
    return nil, errors.New("Failed fetching admin identity")
  }

  ok, err := t.isCaller(stub, adminCertificate)
  if err != nil {
    return nil, errors.New("Failed checking admin identity")
  }
  if !ok {
    return nil, errors.New("The caller is not an administrator")
  }

  // actually create the Asset
  ok, err = stub.InsertRow("AssetsOwnership", shim.Row{
    Columns: []*shim.Column{
      &shim.Column{Value: &shim.Column_String_{String_: asset}},
      &shim.Column{Value: &shim.Column_Bytes{Bytes: owner}}},
  })

  if !ok && err == nil {
    return nil, errors.New("Asset with same name already exists.")
  }

  return nil, err
}

func (t *AssetManagementChaincode) update(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
  if len(args) != 2 {
    return nil, errors.New("Incorrect number of arguments. Expecting 2")
  }

  asset := args[0]
  newOwner, err := base64.StdEncoding.DecodeString(args[1])
  if err != nil {
    return nil, fmt.Errorf("Failed decoding owner")
  }

  // Verify the identity of the caller
  // Only the owner can transfer one of his assets
  var columns []shim.Column
  col1 := shim.Column{Value: &shim.Column_String_{String_: asset}}
  columns = append(columns, col1)

  row, err := stub.GetRow("AssetsOwnership", columns)
  if err != nil {
    return nil, fmt.Errorf("Failed retrieving asset [%s]: [%s]", asset, err)
  }

  prvOwner := row.Columns[1].GetBytes()
  // myLogger.Debugf("Previous owener of [%s] is [% x]", asset, prvOwner)
  if len(prvOwner) == 0 {
    return nil, fmt.Errorf("Invalid previous owner. Nil")
  }

  // Verify ownership
  ok, err := t.isCaller(stub, prvOwner)
  if err != nil {
    return nil, errors.New("Failed checking asset owner identity")
  }
  if !ok {
    return nil, errors.New("The caller is not the owner of the asset")
  }

  // At this point, the proof of ownership is valid, then register transfer
  err = stub.DeleteRow(
    "AssetsOwnership",
    []shim.Column{shim.Column{Value: &shim.Column_String_{String_: asset}}},
  )
  if err != nil {
    return nil, errors.New("Failed deleting row.")
  }

  _, err = stub.InsertRow(
    "AssetsOwnership",
    shim.Row{
      Columns: []*shim.Column{
        &shim.Column{Value: &shim.Column_String_{String_: asset}},
        &shim.Column{Value: &shim.Column_Bytes{Bytes: newOwner}},
      },
    })
  if err != nil {
    return nil, errors.New("Failed inserting row.")
  }

  return nil, nil
}

func (t *AssetManagementChaincode) isCaller(stub *shim.ChaincodeStub, certificate []byte) (bool, error) {
  // In order to enforce access control, we require that the
  // metadata contains the signature under the signing key corresponding
  // to the verification key inside certificate of
  // the payload of the transaction (namely, function name and args) and
  // the transaction binding (to avoid copying attacks)

  // Verify \sigma=Sign(certificate.sk, tx.Payload||tx.Binding) against certificate.vk
  // \sigma is in the metadata

  sigma, err := stub.GetCallerMetadata()
  if err != nil {
    return false, errors.New("Failed getting metadata")
  }
  payload, err := stub.GetPayload()
  if err != nil {
    return false, errors.New("Failed getting payload")
  }
  binding, err := stub.GetBinding()
  if err != nil {
    return false, errors.New("Failed getting binding")
  }

  // myLogger.Debugf("passed certificate [% x]", certificate)
  // myLogger.Debugf("passed sigma [% x]", sigma)
  // myLogger.Debugf("passed payload [% x]", payload)
  // myLogger.Debugf("passed binding [% x]", binding)

  ok, err := stub.VerifySignature(
    certificate,
    sigma,
    append(payload, binding...),
  )
  if err != nil {
    myLogger.Errorf("Failed checking signature [%s]", err)
    return ok, err
  }
  if !ok {
    myLogger.Error("Invalid signature")
  }

  return ok, err
}

// Invoke is called for every transaction: 
// "assign(asset, owner)": to assign ownership of assets. An asset can be owned by a single entity. Can be called only by admin
// "transfer(asset, newOwner)": to transfer the ownership of an asset. Only the owner of the specific asset can call this
// An asset is any string to identify it. An owner is representated by one of his ECert/TCert.
func (t *AssetManagementChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {

  if function == "create" {
    // create asset
    return t.assign(stub, args)
  } else if function == "update" {
    // update asset (transfer ownership etc)
    return t.update(stub, args)
  }

  return nil, errors.New("Received unknown function invocation")
}

// "query(asset)": returns the owner of the asset.
// Anyone can invoke this function.
func (t *AssetManagementChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
  var err error

  if len(args) != 1 {
    return nil, errors.New("Incorrect number of arguments. Expecting name of an asset to query")
  }

  // Who is the owner of the asset?
  asset := args[0]

  var columns []shim.Column
  col1 := shim.Column{Value: &shim.Column_String_{String_: asset}}
  columns = append(columns, col1)

  row, err := stub.GetRow("AssetsOwnership", columns)
  if err != nil {
    return nil, fmt.Errorf("Failed retrieving asset [%s]: [%s]", string(asset), err)
  }

  return row.Columns[1].GetBytes(), nil
}


func main() {
  err := shim.Start(new(AssetManagementChaincode))
  if err != nil {
    fmt.Printf("Error starting Asset Management Chaincode: %s", err)
  }
}