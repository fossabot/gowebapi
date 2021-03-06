package component

import (
	"app/webapi/internal/bind"
	"app/webapi/internal/response"
	"app/webapi/internal/testutil"
	"app/webapi/pkg/query"
)

// NewCoreMock returns all mocked dependencies.
func NewCoreMock() (Core, *CoreMock) {
	ml := new(testutil.MockLogger)
	md := testutil.ConnectDatabase(true)
	mq := query.New(md)
	mt := new(testutil.MockToken)
	resp := response.New()
	binder := bind.New()

	core := NewCore(ml, md, mq, binder, resp, mt)
	m := &CoreMock{
		Log:      ml,
		DB:       md,
		Q:        mq,
		Bind:     binder,
		Response: resp,
		Token:    mt,
	}
	return core, m
}

// CoreMock contains all the mocked dependencies.
type CoreMock struct {
	Log      *testutil.MockLogger
	DB       IDatabase
	Q        IQuery
	Bind     IBind
	Response IResponse
	Token    *testutil.MockToken
}
