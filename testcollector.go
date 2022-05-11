package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"go.mongodb.org/mongo-driver/bson"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// const DSTEST_SUITES_API_URL = "http://localhost:8000/dstestapi/testsuites/"
const DSTEST_RUNS_API_URL = "http://localhost:8000/dstestapi/testruns/"

// const TEST_SUITE_ID = "627a8285b1c63cf751cfc1fd"
const TESTRUN_ID = "627ab98d12367842f1f69560"

// const TESTRUN_HEADER_ID = "1234567890123456"

// Connect to database; get client, context and CancelFunc back
func connect(uri string) (*mongo.Client, context.Context, context.CancelFunc, error) {

	// ctx is used to set db query timeout
	ctx, cancel := context.WithTimeout(context.Background(),
		30*time.Second)

	// connect to db, get client back
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

// Returns transactions for a given test run, ***in descending chronological order***
func findTransactionsForTestRun(testRun TestRun) ([]Transaction, error) {

	// JTE TODO filter also on ApiKey

	// Sort by `Timestamp` field descending
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{"_id", -1}})

	filter := bson.D{{"testrunid", bson.D{{"$eq", testRun.TestRunHeaderId}}}}

	cursor, err := txCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}

	var transactions []Transaction
	if err = cursor.All(ctx, &transactions); err != nil {
		return nil, err
	}

	return transactions, nil
}

// Check whether all the specified predicates match the given transaction
func validatePredicatesForTransaction(predicates []*TestCasePredicate, transaction Transaction) bool {

	// Iterate through all predicates in this test case
	allMatched := true
	for _, predicate := range predicates {
		if !matchPredicate(transaction.Request, predicate.Attribute, predicate.ExpectedValue) {
			allMatched = false
			break
		}
	}

	return allMatched
}

// Find the most recent transaction that matches this test case
func matchTransactionToTestCase(testCase TestCase, transactions []Transaction) (Transaction, TestStatus, error) {

	var err error
	var matchingTransaction Transaction
	testStatus := UndefinedTestStatus

	for _, transaction := range transactions {

		// Check whether URL matches
		if transaction.Url != testCase.Url {
			continue // nope - this is not the transaction we want
		}

		// So far, so good - now attempt to match all predicates
		if validatePredicatesForTransaction(testCase.Predicates, transaction) {

			if transaction.Status == testCase.ExpectedStatus {
				testStatus = Success
			} else {
				testStatus = Failure
			}

			// This transaction matches all checks, so don't need to keep looking
			matchingTransaction = transaction
			break
		}
	}

	// If test status is still undefined, then we did not find a matching transaction
	if testStatus == UndefinedTestStatus {
		err = errors.New("Could not find transaction to match test case " + testCase.Name)
	}

	return matchingTransaction, testStatus, err
}

// Match transactions to each test case in this test run instance
func matchTransactionsToTestRun(testRun *TestRun, transactions []Transaction) error {

	testRunStatus := Complete

	// For each test case in the test suite associated with this run...
	for _, testCase := range testRun.TestSuite.TestCases {

		fmt.Println("Searching for transactions to match test case", testCase.Name)
		transaction, testStatus, err := matchTransactionToTestCase(*testCase, transactions)

		if err == nil {
			fmt.Println("Found matching transaction")
			testResult := TestResult{ /*TestRun: testRun, */ TestCase: testCase, Status: testStatus, Transaction: &transaction, Timestamp: time.Now()}
			testRun.TestResults = append(testRun.TestResults, &testResult)
		} else {
			// Did not find a transaction to match this test case
			fmt.Println("Did not find matching transaction")
			testRunStatus = InProgress
		}
	}

	testRun.Status = testRunStatus

	return nil
}

func cleanse(input string) string {
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

/*
func fetchTestSuite(suiteId string) TestSuite {

	// Invoke the GET testsuite endpoint in the DSTest API
	response, err := http.Get(DSTEST_SUITES_API_URL + suiteId)
	if err != nil {
		log.Fatalln(err)
	}

	//TODO JTE check whether we got a 404 or 200

	// Read the response
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(responseData))

	// Unmarshal the response data into a TestSuite instance
	var testSuite TestSuite
	json.Unmarshal(responseData, &testSuite)

	return testSuite
}
*/

func fetchTestRun(testRunId string) TestRun {

	// Invoke the GET test run endpoint in the DSTest API
	response, err := http.Get(DSTEST_RUNS_API_URL + testRunId)
	if err != nil {
		log.Fatalln(err)
	}

	//TODO JTE check whether we got a 404 or 200

	// Read the response
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(responseData))

	// Unmarshal the response data into a TestSuite instance
	var testRun TestRun
	json.Unmarshal(responseData, &testRun)

	return testRun
}

func collectTestRun(testRunId string) (TestRun, error) {

	// Load the test suite associated with this testRunId from config db
	// testSuite := fetchTestSuite(DSTEST_RUNS_API_URL)

	// Load the test run with the specified id from db
	testRun := fetchTestRun(testRunId)

	/*
		predicate1 := TestCasePredicate{Attribute: "amount.total", ExpectedValue: "300"}
		predicate2 := TestCasePredicate{Attribute: "source.sourceType", ExpectedValue: "PaymentTrack"}
		predicate3 := TestCasePredicate{Attribute: "transactionDetails.captureFlag", ExpectedValue: "True"}
		predicate4 := TestCasePredicate{Attribute: "amount.total", ExpectedValue: "200"}
		predicate5 := TestCasePredicate{Attribute: "amount.total", ExpectedValue: "900"}
		predicate6 := TestCasePredicate{Attribute: "amount.total", ExpectedValue: "600"}
		predicate7 := TestCasePredicate{Attribute: "amount.total", ExpectedValue: "999"}
		testCase1 := TestCase{Id: primitive.NewObjectID(), Name: "TestCase1", Predicates: []*TestCasePredicate{&predicate1, &predicate2, &predicate3}, ExpectedStatus: 201, Url: "/ch/payments/v1/charges"}
		testCase2 := TestCase{Id: primitive.NewObjectID(), Name: "TestCase2", Predicates: []*TestCasePredicate{&predicate4}, ExpectedStatus: 500, Url: "/ch/payments/v1/charges"}
		testCase3 := TestCase{Id: primitive.NewObjectID(), Name: "TestCase3", Predicates: []*TestCasePredicate{&predicate5}, ExpectedStatus: 201, Url: "/ch/payments/v1/charges"}
		testCase4 := TestCase{Id: primitive.NewObjectID(), Name: "TestCase4", Predicates: []*TestCasePredicate{&predicate6}, ExpectedStatus: 201, Url: "/ch/payments/v1/charges"}
		testCase5 := TestCase{Id: primitive.NewObjectID(), Name: "TestCase5", Predicates: []*TestCasePredicate{&predicate7}, ExpectedStatus: 201, Url: "/ch/payments/v1/charges"}
		testSuite1 := TestSuite{Name: "TestSuite1", TestCases: []*TestCase{&testCase1, &testCase2, &testCase3, &testCase4, &testCase5}}
	*/
	//testRun := TestRun{Name: "Tom Run 1", ApiKey: "apikey000001", TestRunHeaderId: testRunHeaderId, TestSuite: &testSuite, Status: InProgress, Timestamp: time.Now()}

	transactions, err := findTransactionsForTestRun(testRun)
	if err != nil {
		return TestRun{}, err
	}

	err = matchTransactionsToTestRun(&testRun, transactions)

	return testRun, err
}

// JTE WARNING: This is super inefficient
func getTestResultforTestCase(testCase TestCase, testRun TestRun) (TestResult, error) {

	// Find the test result matching the specified Test Case
	for _, testResult := range testRun.TestResults {
		if testResult.TestCase.Id == testCase.Id {
			return *testResult, nil
		}

	}

	return TestResult{}, errors.New("Could not find a test result for test case " + testCase.Name)
}

func dumpTestRunReport(testRun TestRun) {

	fmt.Println("Report for Test Run>", testRun.Name)

	for _, testCase := range testRun.TestSuite.TestCases {
		testResult, err := getTestResultforTestCase(*testCase, testRun)
		if err == nil {
			if testResult.Status == Success {
				fmt.Printf("Result for Test Case `%s`: success\n", testCase.Name)
			} else {
				fmt.Printf("Result for Test Case `%s`: failure\n", testCase.Name)
			}
		} else {
			fmt.Printf("Result for Test Case `%s`: no transaction found\n", testCase.Name)
		}
	}
}

func persistTestRun(testRun TestRun) error {

	_, err := testRunsCollection.InsertOne(context.TODO(), testRun)

	return err
}

// Globals
// JTE TODO is there something better to do with these?
var client *mongo.Client
var ctx context.Context
var cancel context.CancelFunc
var db *mongo.Database
var txCollection *mongo.Collection
var testRunsCollection *mongo.Collection

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
	testRunsCollection = db.Collection("testruns")
	fmt.Println("Initialized db and collections")

	// Prepare to collect results for a given TestRun
	testRun, err := collectTestRun(TESTRUN_ID)
	if err != nil {
		fmt.Println("Error during run collection", err)
		panic(err)
	}

	// Persist the test run to db
	persistTestRun(testRun)

	dumpTestRunReport(testRun)
}
