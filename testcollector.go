package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Transaction struct {
	Id        primitive.ObjectID `json:"id,omitempty"`
	ApiKey    string             `json:"apikey,omitempty" validate:"required"`
	TestRunId string             `json:"testrunid,omitempty" validate:"required"`
	Status    int                `json:"status,omitempty" validate:"required"`
	Url       string             `json:"url,omitempty" validate:"required"`
	Headers   string             `json:"headers,omitempty" validate:"required"`
	Request   string             `json:"request,omitempty" validate:"required"`
	Response  string             `json:"response,omitempty" validate:"required"`
	Timestamp time.Time          `json:"timestamp,omitempty" validate:"required"`
}

type TestCasePredicate struct {
	Id            primitive.ObjectID `json:"id,omitempty"`
	Attribute     string             `json:"attribute,omitempty" validate:"required"`
	ExpectedValue string             `json:"expected_value,omitempty" validate:"required"`
}

type TestCase struct {
	Id             primitive.ObjectID  `json:"id,omitempty"`
	Name           string              `json:"name,omitempty" validate:"required"`
	Url            string              `json:"url,omitempty" validate:"required"`
	Predicates     []TestCasePredicate `json:"predicates,omitempty" validate:"required"`
	ExpectedStatus int                 `json:"expected_status,omitempty" validate:"required"`
}

type TestSuite struct {
	Id        primitive.ObjectID `json:"id,omitempty"`
	Name      string             `json:"name,omitempty" validate:"required"`
	TestCases []TestCase         `json:"test_cases,omitempty" validate:"required"`
}

type TestRunStatus int

const (
	UndefinedRunStatus TestRunStatus = iota
	Created
	InProgress
	Complete
)

type TestRun struct {
	Id              primitive.ObjectID `json:"id,omitempty"`
	ApiKey          string             `json:"apikey,omitempty" validate:"required"`
	TestRunHeaderId string             `json:"header_id,omitempty" validate:"required"`
	TestSuite       TestSuite          `json:"test_suite,omitempty" validate:"required"`
	TestResults     []TestResult       `json:"test_results,omitempty" validate:"required"`
	Status          TestRunStatus      `json:"status,omitempty" validate:"required"`
	Timestamp       time.Time          `json:"timestamp,omitempty" validate:"required"`
}

type TestStatus int

const (
	UndefinedTestStatus TestStatus = iota
	Success
	Failure
)

type TestResult struct {
	Id        primitive.ObjectID `json:"id,omitempty"`
	TestRun   TestRun            `json:"test_run,omitempty" validate:"required"`
	TestCase  TestCase           `json:"test_case,omitempty" validate:"required"`
	Status    TestStatus         `json:"status,omitempty" validate:"required"`
	Timestamp time.Time          `json:"timestamp,omitempty" validate:"required"`
}

// Connect to database; get client, context and CancelFunc back
func connect(uri string) (*mongo.Client, context.Context, context.CancelFunc, error) {

	// ctx will be used to set deadline for process, here
	// deadline will of 30 seconds.
	ctx, cancel := context.WithTimeout(context.Background(),
		30*time.Second)

	// mongo.Connect return mongo.Client method
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	return client, ctx, cancel, err
}

// Closes mongoDB connection and cancel context.
func close(client *mongo.Client, ctx context.Context,
	cancel context.CancelFunc) {

	// CancelFunc to cancel to context
	defer cancel()

	// client provides a method to close
	// a mongoDB connection.
	defer func() {

		// client.Disconnect method also has deadline.
		// returns error if any,
		if err := client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
}

/*
func recordTransaction(headers http.Header, status int, request []byte, response []byte) error {

	tx := Transaction{
		Id:        primitive.NewObjectID(),
		ApiKey:    headers.Get("Api-Key"),
		TestRunId: headers.Get("X-TESTRUN-ID"),
		Status:    status,
		Request:   string(request),
		Response:  string(response),
		Timestamp: time.Now(),
	}

	result, err := txCollection.InsertOne(ctx, tx)
	if err != nil {
		fmt.Println("Error inserting document %v", err)
	} else {
		fmt.Println("Inserted document %v", result)
	}

	return nil
}
*/

func findTransactionsForTestRun(testRun TestRun) ([]Transaction, error) {

	// JTE TODO filter also on ApiKey
	filter := bson.D{{"testrunid", bson.D{{"$eq", testRun.TestRunHeaderId}}}}

	cursor, err := txCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	var transactions []Transaction
	if err = cursor.All(ctx, &transactions); err != nil {
		return nil, err
	}
	return transactions, nil
}

func matchTransactionsForTestRun(testRun *TestRun, transactions []Transaction) error {

	for _, transaction := range transactions {
		fmt.Println(">>>>>>>>>>>>>>>>>>>>>")
		matchTransactionForTestRun(testRun, transaction)
		fmt.Println("<<<<<<<<<<<<<<<<<<<<<")
	}
	return nil
}

/*
func test1(transaction Transaction) bool {

	return matchPredicate(transaction.Request, "amount.total", "100") &&
		matchPredicate(transaction.Request, "source.sourceType", "PaymentTrack") &&
		matchPredicate(transaction.Request, "transactionDetails.captureFlag", "True")

}

func test2(transaction Transaction) bool {

	return matchPredicate(transaction.Request, "amount.total", "200") &&
		matchPredicate(transaction.Request, "source.sourceType", "PaymentTrack") &&
		matchPredicate(transaction.Request, "transactionDetails.captureFlag", "True")
}
*/

func matchTransactionForTestRun(testRun *TestRun, transaction Transaction) error {

	/*
		// Load the test suite associated with this testRunId
		predicate1 := TestCasePredicate{Attribute: "amount.total", ExpectedValue: "300"}
		predicate2 := TestCasePredicate{Attribute: "source.sourceType", ExpectedValue: "PaymentTrack"}
		predicate3 := TestCasePredicate{Attribute: "transactionDetails.captureFlag", ExpectedValue: "True"}
		predicate4 := TestCasePredicate{Attribute: "amount.total", ExpectedValue: "200"}
		testCase1 := TestCase{Name: "TestCase1", Predicates: []TestCasePredicate{predicate1, predicate2, predicate3}, ExpectedStatus: 201, Url: "/ch/payments/v1/charges"}
		testCase2 := TestCase{Name: "TestCase2", Predicates: []TestCasePredicate{predicate4}, ExpectedStatus: 500, Url: "/ch/payments/v1/charges"}
		testSuite1 := TestSuite{Name: "TestSuite1", TestCases: []TestCase{testCase1, testCase2}}
	*/
	//fmt.Println(testSuite1)

	// Iterate through all test cases in this suite
	for _, testCase := range testRun.TestSuite.TestCases {

		fmt.Printf("---Checking transaction against test case %s\n", testCase.Name)

		// Check that the Url matches
		if testCase.Url != transaction.Url {
			fmt.Println("Url does not match")
			continue // no match, so skip this test case
		} else {
			fmt.Println("Url does match")
		}

		// Iterate through all predicates in this test case
		allMatched := true
		for _, predicate := range testCase.Predicates {
			if !matchPredicate(transaction.Request, predicate.Attribute, predicate.ExpectedValue) {
				//fmt.Println("Predicate does not match")
				allMatched = false
				break
			} else {
				//fmt.Println("Predicate does match")
			}

		}
		if allMatched {
			fmt.Printf("Test case %s matches all %d predicates for transaction %v\n", testCase.Name, len(testCase.Predicates), transaction.Id)

			// We found the right test case matching this transaction, now determine whether the http response
			// matches what is considered success for this test case
			testStatus := UndefinedTestStatus
			if testCase.ExpectedStatus == transaction.Status {
				fmt.Printf("*** Successful test case\n")
				testStatus = Success
			} else {
				fmt.Printf("*** Failed test case\n")
				testStatus = Failure
			}

			testResult := TestResult{TestRun: *testRun, TestCase: testCase, Status: testStatus, Timestamp: time.Now()}
			testRun.TestResults = append(testRun.TestResults, testResult)
			return nil // we're done with this call since we matched the transaction to a test case
		} else {
			fmt.Printf("Test case %s does not match all %d predicates\n", testCase.Name, len(testCase.Predicates))

		}

	}
	/*
		if test1(transaction) {
			fmt.Println("Transaction matches Test1")
		} else if test2(transaction) {
			fmt.Println("Transaction matches Test2")
		} else {
			fmt.Println("Unmatched transaction")
		}
	*/
	return nil
}

func cleanse(input string) string {

	// foo := strings.Trim(input, "\"")
	// fmt.Println(foo)
	return strings.Trim(input, "\"")
}

func matchPredicate(requestString string, predicate string, expectedValue string) bool {

	value := gjson.Get(requestString, predicate)

	if value.Raw == "" {
		return false //, errors.New("Predicate not found")
	}

	if strings.ToLower(cleanse(value.Raw)) == strings.ToLower(expectedValue) {
		return true //, nil
	} else {
		return false //, errors.New("Expected value not matched")
	}
	// Should never get here
}

func collectTestRun(testRunHeaderId string) (TestRun, error) {

	// Load the test suite associated with this testRunId
	predicate1 := TestCasePredicate{Attribute: "amount.total", ExpectedValue: "300"}
	predicate2 := TestCasePredicate{Attribute: "source.sourceType", ExpectedValue: "PaymentTrack"}
	predicate3 := TestCasePredicate{Attribute: "transactionDetails.captureFlag", ExpectedValue: "True"}
	predicate4 := TestCasePredicate{Attribute: "amount.total", ExpectedValue: "200"}
	testCase1 := TestCase{Name: "TestCase1", Predicates: []TestCasePredicate{predicate1, predicate2, predicate3}, ExpectedStatus: 201, Url: "/ch/payments/v1/charges"}
	testCase2 := TestCase{Name: "TestCase2", Predicates: []TestCasePredicate{predicate4}, ExpectedStatus: 500, Url: "/ch/payments/v1/charges"}
	testSuite1 := TestSuite{Name: "TestSuite1", TestCases: []TestCase{testCase1, testCase2}}
	testRun := TestRun{ApiKey: "apikey000001", TestRunHeaderId: testRunHeaderId, TestSuite: testSuite1, Status: UndefinedRunStatus, Timestamp: time.Now()}

	transactions, err := findTransactionsForTestRun(testRun)
	if err != nil {
		return TestRun{}, err
	}

	err = matchTransactionsForTestRun(&testRun, transactions)

	return testRun, err
}

// Globals
// JTE TODO is there something better to do with these?
var client *mongo.Client
var ctx context.Context
var cancel context.CancelFunc
var db *mongo.Database
var txCollection *mongo.Collection

func main() {

	// Initialize database (hardcoded for local machine)
	client, ctx, cancel, err := connect("mongodb://localhost:27017")
	if err != nil {
		panic(err)
	}

	// Close db when the main function is returned.
	defer close(client, ctx, cancel)
	fmt.Println("Connected to local mongodb")

	// Get target database and collection
	db = client.Database("dstest")
	txCollection = db.Collection("transactions")
	fmt.Println("Initialized db and collection")

	// Prepare to collect results for a given TestRun
	testRun, err := collectTestRun("1234567890123456")
	if err != nil {
		fmt.Println("Error during run collection", err)
		panic(err)
	}
	fmt.Println(testRun.Status)
}

