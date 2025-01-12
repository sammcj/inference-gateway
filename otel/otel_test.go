package otel_test

import (
	"errors"
	"testing"

	config "github.com/edenreich/inference-gateway/config"
	mocks "github.com/edenreich/inference-gateway/otel/mocks"
	assert "github.com/stretchr/testify/assert"
	trace "go.opentelemetry.io/otel/sdk/trace"
	gomock "go.uber.org/mock/gomock"
)

func TestOpenTelemetryImpl_Init_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOTEL := mocks.NewMockOpenTelemetry(ctrl)

	cfg := config.Config{
		Environment:     "development",
		ApplicationName: "TestApp",
	}

	mockOTEL.EXPECT().Init(cfg).Return(&trace.TracerProvider{}, nil)

	tp, err := mockOTEL.Init(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, tp)
}

func TestOpenTelemetryImpl_Init_Failure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOTEL := mocks.NewMockOpenTelemetry(ctrl)

	cfg := config.Config{
		Environment:     "production",
		ApplicationName: "TestApp",
	}

	mockOTEL.EXPECT().Init(cfg).Return(nil, errors.New("initialization error"))

	tp, err := mockOTEL.Init(cfg)
	assert.Error(t, err)
	assert.Nil(t, tp)
}
