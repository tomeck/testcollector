package main

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Transaction struct {
	Id        primitive.ObjectID `json:"id,omitempty"`
	ApiKey    string             `json:"apikey,omitempty" validate:"required"`
	TestRun   *TestRun           `json:"test_run,omitempty" validate:"required"`
	Status    int                `json:"status,omitempty" validate:"required"`
	Url       string             `json:"url,omitempty" validate:"required"`
	Headers   string             `json:"headers,omitempty" validate:"required"`
	Request   string             `json:"request,omitempty" validate:"required"`
	Response  string             `json:"response,omitempty" validate:"required"`
	Timestamp time.Time          `json:"timestamp,omitempty" validate:"required"`
}

type TestCasePredicate struct {
	Id            primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Name          string             `json:"name,omitempty" validate:"required"`
	Attribute     string             `json:"attribute,omitempty" validate:"required"`
	ExpectedValue string             `json:"expected_value,omitempty" validate:"required"`
}

type TestCase struct {
	Id             primitive.ObjectID   `json:"_id,omitempty" bson:"_id,omitempty"`
	Name           string               `json:"name,omitempty" validate:"required"`
	Url            string               `json:"url,omitempty" validate:"required"`
	Predicates     []*TestCasePredicate `json:"predicates,omitempty" validate:"required"`
	ExpectedStatus int                  `json:"expected_status,omitempty" validate:"required"`
}

type TestSuite struct {
	Id        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Name      string             `json:"name,omitempty" validate:"required"`
	TestCases []*TestCase        `json:"test_cases,omitempty" validate:"required"`
}

type TestRunStatus int

const (
	UndefinedRunStatus TestRunStatus = iota
	Created
	InProgress
	Complete
)

type TestRun struct {
	Id              primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Name            string             `json:"name,omitempty" validate:"required"`
	ApiKey          string             `json:"apikey,omitempty" validate:"required"`
	TestRunHeaderId string             `json:"header_id,omitempty" validate:"required"`
	TestSuite       *TestSuite         `json:"test_suite,omitempty" validate:"required"`
	TestResults     []*TestResult      `json:"test_results,omitempty" validate:"required"`
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
	Id primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	//TestRun     *TestRun           `json:"test_run,omitempty" validate:"required"`
	TestCase    *TestCase    `json:"test_case,omitempty" validate:"required"`
	Transaction *Transaction `json:"transaction,omitempty" validate:"required"`
	Status      TestStatus   `json:"status,omitempty" validate:"required"`
	Timestamp   time.Time    `json:"timestamp,omitempty" validate:"required"`
}
