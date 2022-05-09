package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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
func validatePredicatesForTransaction(predicates []TestCasePredicate, transaction Transaction) bool {

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

/*
func matchTransactionsForTestRun(testRun *TestRun, transactions []Transaction) error {

	for _, transaction := range transactions {
		fmt.Println(">>>>>>>>>>>>>>>>>>>>>")
		matchTransactionForTestRun(testRun, transaction)
		fmt.Println("<<<<<<<<<<<<<<<<<<<<<")
	}
	return nil
}
*/

// Match transactions to each test case in this test run instance
func matchTransactionsToTestRun(testRun *TestRun, transactions []Transaction) error {

	testRunStatus := Complete

	// For each test case in the test suite associated with this run...
	for _, testCase := range testRun.TestSuite.TestCases {

		fmt.Println("Searching for transactions to match test case", testCase.Name)
		transaction, testStatus, err := matchTransactionToTestCase(testCase, transactions)

		if err == nil {
			fmt.Println("Found matching transaction")
			testResult := TestResult{TestRun: *testRun, TestCase: testCase, Status: testStatus, Transaction: transaction, Timestamp: time.Now()}
			testRun.TestResults = append(testRun.TestResults, testResult)
		} else {
			// Did not find a transaction to match this test case
			fmt.Println("Did not find matching transaction")
			testRunStatus = InProgress
		}
	}

	testRun.Status = testRunStatus

	return nil
}

/*
func testSatisfied(testRun TestRun, testCase TestCase) bool {
	retval := false

	for _, testResult := range testRun.TestResults {

		if testResult.TestCase.Id == testCase.Id && testResult.Status == Success {
			retval = true
			break
		}
	}

	return retval
}
*/

/*
func matchTransactionForTestRun(testRun *TestRun, transaction Transaction) error {

	// Iterate through all test cases in this suite
	for _, testCase := range testRun.TestSuite.TestCases {

		// Check whether we already have a successful transaction against this test case
		if testSatisfied(*testRun, testCase) {
			fmt.Println("Test case already satisfied")
			continue
		}
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

	return nil
}
*/

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

func collectTestRun(testRunHeaderId string) (TestRun, error) {

	// Load the test suite associated with this testRunId from config db
	predicate1 := TestCasePredicate{Attribute: "amount.total", ExpectedValue: "300"}
	predicate2 := TestCasePredicate{Attribute: "source.sourceType", ExpectedValue: "PaymentTrack"}
	predicate3 := TestCasePredicate{Attribute: "transactionDetails.captureFlag", ExpectedValue: "True"}
	predicate4 := TestCasePredicate{Attribute: "amount.total", ExpectedValue: "200"}
	predicate5 := TestCasePredicate{Attribute: "amount.total", ExpectedValue: "900"}
	predicate6 := TestCasePredicate{Attribute: "amount.total", ExpectedValue: "600"}
	predicate7 := TestCasePredicate{Attribute: "amount.total", ExpectedValue: "999"}
	testCase1 := TestCase{Id: primitive.NewObjectID(), Name: "TestCase1", Predicates: []TestCasePredicate{predicate1, predicate2, predicate3}, ExpectedStatus: 201, Url: "/ch/payments/v1/charges"}
	testCase2 := TestCase{Id: primitive.NewObjectID(), Name: "TestCase2", Predicates: []TestCasePredicate{predicate4}, ExpectedStatus: 500, Url: "/ch/payments/v1/charges"}
	testCase3 := TestCase{Id: primitive.NewObjectID(), Name: "TestCase3", Predicates: []TestCasePredicate{predicate5}, ExpectedStatus: 201, Url: "/ch/payments/v1/charges"}
	testCase4 := TestCase{Id: primitive.NewObjectID(), Name: "TestCase4", Predicates: []TestCasePredicate{predicate6}, ExpectedStatus: 201, Url: "/ch/payments/v1/charges"}
	testCase5 := TestCase{Id: primitive.NewObjectID(), Name: "TestCase5", Predicates: []TestCasePredicate{predicate7}, ExpectedStatus: 201, Url: "/ch/payments/v1/charges"}
	testSuite1 := TestSuite{Name: "TestSuite1", TestCases: []TestCase{testCase1, testCase2, testCase3, testCase4, testCase5}}
	testRun := TestRun{Name: "Tom Run 1", ApiKey: "apikey000001", TestRunHeaderId: testRunHeaderId, TestSuite: testSuite1, Status: InProgress, Timestamp: time.Now()}

	transactions, err := findTransactionsForTestRun(testRun)
	if err != nil {
		return TestRun{}, err
	}

	/*
		err = matchTransactionsForTestRun(&testRun, transactions)

		// Check if all test cases have been satisfied
		if len(testRun.TestResults) == len(testRun.TestSuite.TestCases) {
			testRun.Status = Complete
		}
	*/

	err = matchTransactionsToTestRun(&testRun, transactions)

	return testRun, err
}

// JTE WARNING: This is super inefficient
func getTestResultforTestCase(testCase TestCase, testRun TestRun) (TestResult, error) {

	// Find the test result matching the specified Test Case
	for _, testResult := range testRun.TestResults {
		if testResult.TestCase.Id == testCase.Id {
			return testResult, nil
		}

	}

	return TestResult{}, errors.New("Could not find a test result for test case " + testCase.Name)
}

func dumpTestRunReport(testRun TestRun) {

	fmt.Println("Report for Test Run>", testRun.Name)

	for _, testCase := range testRun.TestSuite.TestCases {
		testResult, err := getTestResultforTestCase(testCase, testRun)
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

	/*
		for _, testResult := range testRun.TestResults {
			fmt.Printf("Result for Test Case `%s`: %d\n", testResult.TestCase.Name, testResult.Status)
		}

				for _, testCase := range testRun.TestSuite.TestCases {
				testResult, err := getTestResultforTestCase(testCase, testRun)
				if err == nil {
					fmt.Printf("Result for Test Case `%s`: %d\n", testCase.Name, testResult.Status)
				} else {
					fmt.Println(err)
				}
			}

		fmt.Printf("Status of this test run: %d", testRun.Status)

		// for _, testResult := range testRun.TestResults {
		// 	fmt.Printf("Result for Test Case `%s`: %d\n", testResult.TestCase.Name, testResult.Status)
		// }
	*/
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

	dumpTestRunReport(testRun)
}
