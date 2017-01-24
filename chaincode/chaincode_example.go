/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	"fmt"
	"strconv"
	"encoding/json"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

var contractIndexStr = "_contractindex"		// name for key/value that will store the list of contracts,
																					// contract ID as key
var shipmentIndexStr = "_shipmentindex"		// name for key/value that will store the list of shipments
																					// contract ID as key
type ContractTerms struct{
	Max_Temperature_F int `json:"max_temperature_f"`
	Product_Type string `json:"product"`
	ContractID string `json:"contractID"`
}

type Party struct{
	Company string `json:"company"`
	Bank		string `json:"bank"`
}

type LetterOfCredit struct{
	ContractID	string `json:"contractID"`
	Description string `json:"description"`
	Value				int `json:"value_dollars"`
	Importer		Party `json:"importer"`
	Exporter		Party `json:"exporter"`
	ShippingCo	string `json:"shipping_co"`
	Customs			string `json:"customs_auth"`
	PortOfLoad	string `json:"port_of_loading"`
	PortOfEntry string `json:"port_of_entry"`
	Timestamp			int64 `json:"timestamp"`
}

type Shipment struct{
	ContractID		string `json:"contractID"`
	Value					int `json:"value_dollars"`
	Start_Temp_F	int `json:"start_temp_f"`
	End_Temp_F		int `json:"end_temp_f"`
	CarrierName		string `json:"carrier_name"`
	Location			string `json:"location"`
	ShipEvent			string `json:"shipEvent"`
	Timestamp			int64 `json:"timestamp"`
}

var EVENT_COUNTER = "event_counter"

// ================================================================================
// Main
// ================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Supply Chain chaincode: %s", err)
	}
}

// ================================================================================
// Run - Our entry point for Invocations 
// ================================================================================

func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	var Aval int // Asset holdings
	var err error

	fmt.Printf("Called Init()")
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	// Initialize the chaincode
	Aval, err = strconv.Atoi(args[0])
	if err != nil {
		return nil, errors.New("Expecting integer value for asset holding")
	}

	fmt.Printf("Initializing abc to %d\n", Aval)

	// Write the state to the ledger
	err = stub.PutState("abc", []byte(strconv.Itoa(Aval)))				//making a test var "abc", I find it handy to read/write to it right away to test the network
	if err != nil {
		return nil, err
	}

	var empty []string
	jsonAsBytes, _ := json.Marshal(empty)								//marshal an emtpy array of strings to clear the index
	err = stub.PutState(contractIndexStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}

	err = stub.PutState(shipmentIndexStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}

  err = stub.PutState(EVENT_COUNTER, []byte("1"))
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "init" {													//initialize the chaincode state, used as reset
		return t.Init(stub, "init", args)
	} else if function == "init_contract_terms" {				//create a business contract 
		return t.init_terms(stub, args)
	} else if function == "create_loc" {
		t.create_letter_of_credit(stub, args)
	} else if function == "shipment_activity" {
		t.shipment_activity(stub, args)
	} else{
	/*else if function == "shipment_event" {				//Enter the shipment event within the supply chain route 
		return t.shipment_event(stub, args)
	} else if function == "transfer_funds" {		  //transfer funds from one participant to another
		res, err := t.transfer_funds(stub, args)
		//cleanTrades(stub)													//lets make sure all open trades are still valid
		return res, err
	}*/
		fmt.Println("invoke did not find func: " + function)					//error
	}
	//Event based
  b, err := stub.GetState(EVENT_COUNTER)
	if err != nil {
		return nil, errors.New("Failed to get state")
	}
	noevts, _ := strconv.Atoi(string(b))

	tosend := "Event Counter is " + string(b)

	err = stub.PutState(EVENT_COUNTER, []byte(strconv.Itoa(noevts+1)))
	if err != nil {
		return nil, err
	}

	err = stub.SetEvent("evtsender", []byte(tosend))
	if err != nil {
		return nil, err
  }

	return nil, errors.New("Received unknown function invocation")


}

// Transaction makes payment of X units from A to B
func (t *SimpleChaincode) init_terms(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	var err error

	fmt.Printf("init_terms(): Initializing a contract with args: %d\n", len(args))
	// 0								1							2
	// "contract_id" "product_type"	"max_temperature_f"
	if len(args) != 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 3")
	}

	//input sanitation
	if len(args[0]) <= 0 {
		return nil, errors.New("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return nil, errors.New("2nd argument must be a number")
	}
	if len(args[2]) <= 0 {
		return nil, errors.New("3nd argument must be a non-empty string")
	}

	fmt.Println("- This start init contract terms")
	// Get input args
	contract_id				:= args[0]				// Contract ID
	fmt.Println("contract_id: %s\n", contract_id)

	product_type			:= strings.ToLower(args[1])			// type of product being transferred  
	fmt.Println("product_type: %s\n", product_type)

	if err != nil {
		return nil, errors.New("2nd argument must be a numeric string")
	}

	max_temperature_f, err := strconv.Atoi(args[2])	// max temperature
	fmt.Println("max_temperature: %d\n", max_temperature_f)

	// Get the state from the ledger
	contractAsBytes, err := stub.GetState(contract_id)
	if err != nil {
		return nil, errors.New("Failed to get state")
	}

	res := ContractTerms{}
	json.Unmarshal(contractAsBytes, &res)
	if res.ContractID == contract_id {
		retstr := "Terms of Contract for product " + res.ContractID + " already exists"
		return nil, errors.New(retstr)
	}

	//build the contract json string manually
	str := `{"contract_ID": "` + contract_id + `", "product_type": "` + product_type + `", "max_temperature_f": "` + strconv.Itoa(max_temperature_f) + `"}`

	fmt.Printf("Creating new Contract %s\n", str)
	err = stub.PutState(contract_id, []byte(str))						//store contract with contract ID as key
	if err != nil {
		fmt.Printf("ERRORR!\n")
		return nil, err
	}
		//get the contracts index
	contractsAsBytes, err := stub.GetState(contractIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get contract terms index")
	}
	var contractIndex []string
	json.Unmarshal(contractsAsBytes, &contractIndex)							//un stringify it aka JSON.parse()

	//append
	contractIndex = append(contractIndex, product_type)						//add the contract_id to index list
	fmt.Println("! contract index: ", contractIndex)
	jsonAsBytes, _ := json.Marshal(contractIndex)
	err = stub.PutState(contractIndexStr, jsonAsBytes)						//store name of marble

	fmt.Println("- end init contract terms\n")

	//Event based
        b, err := stub.GetState(EVENT_COUNTER)
	if err != nil {
		return nil, errors.New("Failed to get state")
	}
	noevts, _ := strconv.Atoi(string(b))

	tosend := "Event Counter is " + string(b)

	err = stub.PutState(EVENT_COUNTER, []byte(strconv.Itoa(noevts+1)))
	if err != nil {
		return nil, err
	}

	err = stub.SetEvent("evtsender", []byte(tosend))
	if err != nil {
		return nil, err
        }
	return nil, nil
}

// Create a Letter of Credit
func (t *SimpleChaincode) create_letter_of_credit(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	// contractID	desc value_dollars importer exporter shipper customs portOfLoading portOfEntry	
	if len(args) !=  9 {
		return nil, errors.New("Incorrect number of arguments. Expecting 9")
	}

	fmt.Println("Create a Letter Of Credit")
	if len(args[0]) <= 0 {
		return nil, errors.New("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}
	if len(args[3]) <= 0 {
		return nil, errors.New("4th argument must be a non-empty string")
	}
	if len(args[4]) <= 0 {
		return nil, errors.New("5th argument must be a non-empty string")
	}
	if len(args[5]) <= 0 {
		return nil, errors.New("6th argument must be a non-empty string")
	}
	if len(args[6]) <= 0 {
		return nil, errors.New("7th argument must be a non-empty string")
	}
	if len(args[7]) <= 0 {
		return nil, errors.New("8th argument must be a non-empty string")
	}
	if len(args[8]) <= 0 {
		return nil, errors.New("9th argument must be a non-empty string")
	}

	// Decode the json object

//	var loc map[string]interface{}
//  err := json.Unmarshal([]byte(LetterOfCredit), &loc)

  // pull out the parents object
//	parents  := loc["contractID"].(map[string]interface{})
//  if err != nil {
//    panic(err)
//  }

  // Print out mother and father
//   fmt.Printf("Mother: %s\n", u.Parents.Mother)
//   fmt.Printf("Father: %s\n", u.Parents.Father)
	return nil, nil
}

// Enter a shipment activtiy 
func (t *SimpleChaincode) shipment_activity(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	// contractID	value_dollars	start_temp_f	end_temp_f	carrier_name	location	shipEvent
	if len(args) !=  7 {
		return nil, errors.New("Incorrect number of arguments. Expecting 7")
	}

	fmt.Println("Add shipment activity")
	if len(args[0]) <= 0 {
		return nil, errors.New("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return nil, errors.New("3nd argument must be a non-empty string")
	}
	if len(args[3]) <= 0 {
		return nil, errors.New("4nd argument must be a non-empty string")
	}
	if len(args[4]) <= 0 {
		return nil, errors.New("5nd argument must be a non-empty string")
	}
	if len(args[5]) <= 0 {
		return nil, errors.New("6nd argument must be a non-empty string")
	}
	if len(args[6]) <= 0 {
		return nil, errors.New("7nd argument must be a non-empty string")
	}

/*	var shipment Shipment
	var err error

	fmt.Println("shipment arg0: %s\n", args[0])
	shipment.ContractID = args[0]
	shipment.Value, err =  strconv.Atoi(args[1])
	shipment.Start_Temp_F, err=  strconv.Atoi(args[2])
	shipment.End_Temp_F, err =  strconv.Atoi(args[3])
	shipment.CarrierName = args[4]
	shipment.Location = args[5]
	shipment.ShipEvent = args[6]
	shipment.Timestamp = makeTimestamp()
	//activity := []byte(args[0])
	//err = json.Unmarshal(activity, &shipment)
  //if err != nil {
  // fmt.Println("Error in reading JSON\n") 
  //}
//	fmt.Printf("Shipment for contract ID: %s\n", shipment.ContractID)
	// Does contract ID exist
	// Get the state from the ledger
/*	contractAsBytes, err := stub.GetState(shipment.ContractID)
	if err != nil {
		return nil, errors.New("ContractID doesn't exist")
	}

	shipmentAsBytes, err := stub.GetState(shipmentIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get shipement index")
	}
	var shipmenttIndex []string
	json.Unmarshal(shipmentsAsBytes, &shipmentIndex)		//un stringify it aka JSON.parse()


	if res.ContractID == contract_id {
		retstr := "Terms of Contract for product " + res.ContractID + " already exists"
		return nil, errors.New(retstr)
	}

*/
	return nil, nil

}

// Deletes an entity from state
func (t *SimpleChaincode) delete(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	A := args[0]

	// Delete the key from the state in ledger
	err := stub.DelState(A)
	if err != nil {
		return nil, errors.New("Failed to delete state")
	}

	return nil, nil
}

// Query callback representing the query of a chaincode
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	if function != "query" {
		return nil, errors.New("Invalid query function name. Expecting \"query\"")
	}

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the person to query")
	}

	// Get the state from the ledger
	fmt.Printf("Inside Query Response:\n")
	return nil, nil
}

// ========================================================================================================    ====================
// Make Timestamp - create a timestamp in ms
// ========================================================================================================    ====================
func makeTimestamp() int64 {
    return time.Now().UnixNano() / (int64(time.Millisecond)/int64(time.Nanosecond))
}
