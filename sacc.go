/*
 * Copyright IBM Corp All Rights Reserved
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"fmt"
	"os"
	"bytes"
    "bufio"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	GitGoIpfsApi "github.com/ipfs/go-ipfs-api" // GitGoIpfsApi is alias
)

// SimpleAsset implements a simple chaincode to manage an asset
type SimpleAsset struct {
}

// Init is called during chaincode instantiation to initialize any
// data. Note that chaincode upgrade also calls this function to reset
// or to migrate data.
func (t *SimpleAsset) Init(stub shim.ChaincodeStubInterface) peer.Response {
	// Get the args from the transaction proposal
	args := stub.GetStringArgs()
	if len(args) != 2 {
		return shim.Error("Incorrect arguments. Expecting a key and a value")
	}

	// Set up any variables or assets here by calling stub.PutState()

	// We store the key and the value on the ledger
	err := stub.PutState(args[0], []byte(args[1]))
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to create asset: %s", args[0]))
	}
	return shim.Success(nil)
}

// Invoke is called per transaction on the chaincode. Each transaction is
// either a 'get' or a 'set' on the asset created by Init function. The Set
// method may create a new asset by specifying a new key-value pair.
func (t *SimpleAsset) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	// Extract the function and args from the transaction proposal
	fn, args := stub.GetFunctionAndParameters()

	var result string
	var err error
	if fn == "set" {
		result, err = set(stub, args)
	} else if fn == "get" { // assume 'get' even if fn is nil
		result, err = get(stub, args)
	} else if fn == "set_addipfs" {
		result, err = set_addipfs(stub, args)
	} else if fn == "get_catipfs" {
		result, err = get_catipfs(stub, args)
	} else {
		logger.Error("Unsupported function.");
		return shim.Error(err.Error())
	}
	if err != nil {
		return shim.Error(err.Error())
	}

	// Return the result as success payload
	return shim.Success([]byte(result))
}

// Set stores the asset (both key and value) on the ledger. If the key exists,
// it will override the value with the new one
func set(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("Incorrect arguments. Expecting a key and a value")
	}

	err := stub.PutState(args[0], []byte(args[1]))
	if err != nil {
		return "", fmt.Errorf("Failed to set asset: %s", args[0])
	}
	return args[1], nil
}

// Get returns the value of the specified asset key
func get(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("Incorrect arguments. Expecting a key")
	}

	value, err := stub.GetState(args[0])
	if err != nil {
		return "", fmt.Errorf("Failed to get asset: %s with error: %s", args[0], err)
	}
	if value == nil {
		return "", fmt.Errorf("Asset not found: %s", args[0])
	}
	return string(value), nil
}

// ipfs calling 
func set_addipfs(stub shim.ChaincodeStubInterface, args []string) (string, error) {

	var sender, receiver, filename string

	if len(args) != 2 {
		return "", fmt.Errorf("Incorrect arguments. Expecting filename")
	}

	key := args[0]
	logger.Info( "key: " + key )
	value := args[1]
	logger.Info( "value: " + value )

	str_Slice := strings.Split( value, "|")
	logger.Info(str_Slice)

	for i, str_Slice := range str_Slice {
		if i==0 {
			sender = str_Slice
		} else if i == 1  {
			receiver = str_Slice
		} else if i == 2  {
			filename = str_Slice
		}
	}

	logger.Info( key, sender, receiver, filename)

	logger.Info("Add to ipfs: " + filename)

// search with container name (ipfs0)
	mhash, err := AddIpfs( "ipfs0", "5001", filename)

	if err != nil {
		logger.Info("AddIpfs() error")
		jsonResp := "{\"Error\":\"Failed to add to IPFS" + "\"}"
		return "", fmt.Errorf(jsonResp)
	}
    logger.Info( "Success to add on ipfs: " +  mhash)
	value = value + "|" +  mhash

	stub_err := stub.PutState(key, []byte(value))
	if stub_err != nil {
		return "", fmt.Errorf("Failed to set asset: %s", key)
	}
    logger.Info( "Success to set on ledger. key:" +  key + ", value: " + value)

	jsonResp := "{\"Name\":\"" + key  + "\"}"
	fmt.Printf("Query Response:%s\n", jsonResp)

	return key, nil
}

func get_catipfs(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	var sender, receiver, filename, mhash string

	if len(args) != 1 {
		return "", fmt.Errorf("Incorrect arguments. Expecting document number.")
	}
	key :=  args[0]

	logger.Info( "document number(key):" + key )

	value, err := stub.GetState(args[0])
	if err != nil {
		return "", fmt.Errorf("Failed to get asset: %s with error: %s", key, err)
	}
	if value == nil {
		return "", fmt.Errorf("Asset not found: %s", key)
	}

	str_Slice := strings.Split( string(value), "|")
	logger.Info(str_Slice)

	for i, str_Slice := range str_Slice {
		if i==0 {
			sender = str_Slice
		} else if i == 1  {
			receiver = str_Slice
		} else if i == 2  {
			filename = str_Slice
		} else if i == 3 {
			mhash = str_Slice
		}
	}
	logger.Info( "From ledger: key=",  key)
	logger.Info( "From ledger: sender=",  sender)
	logger.Info( "From ledger: receiver=",  receiver)
	logger.Info( "From ledger: filename=",  filename)
	logger.Info( "From ledger: mhash=",  mhash)

	contents, err := CatIpfs( "ipfs0", "5001", mhash)
	if err != nil {
		logger.Info("CatIpfs() error")
        jsonResp := "{\"Error\":\"Failed to add to IPFS" + "\"}"
        return "", fmt.Errorf(jsonResp)
	}

	jsonResp := "{\"contents\":\"" + contents  + "\"}"
    fmt.Printf("Query Response:%s\n", jsonResp)

	return mhash, nil
}

func CatIpfs(Ip string, Port string, mhash string) (string, error) {

	
	UrlPort := Ip + ":" + Port
	shell := GitGoIpfsApi.NewShell(UrlPort)

	reader, err := shell.Cat(mhash)
	if err != nil {
		logger.Error("shell.Cat() error.")
		return mhash, err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	str_buf := buf.String()
	logger.Info("buf: " + str_buf)

	return str_buf, err
}

func AddIpfs(Ip string, Port string, filename string) (string, error) {

	UrlPort := Ip + ":" + Port
	shell := GitGoIpfsApi.NewShell(UrlPort)
	bytedata, err := RetrieveROM( filename )
	if err != nil  {
    	logger.Info("file open error:" + filename);
		return filename, err
	}

	s := string(bytedata[:])
	bufferExample := bytes.NewBufferString(s)

	mhash, err := shell.Add(bufferExample)
	if err != nil {
		logger.Error("shell.Add() error.")
		return filename, err
	}

/*/
	file_mhash = "/ipfs" +  mhash
	buf, err = shell.Cat( file_mhash)
	if err != nil {
		logger.Error("shell.Cat() error.")
		return filename, err
	}
*/
	return mhash, err
}

func AddNoPinIpfs(Ip string, Port string, filename string) (string, error) {

	UrlPort := Ip + ":" + Port
	shell := GitGoIpfsApi.NewShell(UrlPort)
	bytedata, err := RetrieveROM( filename )
	check(err)

	s := string(bytedata[:])
	bufferExample := bytes.NewBufferString(s)

	mhash, err := shell.AddNoPin(bufferExample)

	check(err)

	return mhash, err
}

// Loading data of a file to byte memory
func RetrieveROM(filename string) ([]byte, error) {
    file, err := os.Open(filename)

    if err != nil {
        return nil, err
    }
    defer file.Close()

    stats, statsErr := file.Stat()
    if statsErr != nil {
        return nil, statsErr
    }

    var size int64 = stats.Size()
    bytes := make([]byte, size)

	fmt.Println("file size : ", size);

    bufr := bufio.NewReader(file)
    _,err = bufr.Read(bytes)

    return bytes, err
}

// Reading files requires checking most calls for errors.
// This helper will streamline our error checks below.
func check(e error) {
    if e != nil {
        panic(e)
    }
}
var logger = shim.NewLogger("myChaincode")

// main function starts up the chaincode in the container during instantiate
func main() {
	logger.SetLevel(shim.LogInfo)
    logLevel, _ := shim.LogLevel(os.Getenv("SHIM_LOGGING_LEVEL"))
    shim.SetLoggingLevel(logLevel)

    logger.Info("Gwisang Chaincode");

	if err := shim.Start(new(SimpleAsset)); err != nil {
		fmt.Printf("Error starting SimpleAsset chaincode: %s", err)
	}
}
