package proto6server

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/internal/fwserver"
	"github.com/hashicorp/terraform-plugin-framework/internal/testing/testprovider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestServerConfigureProvider(t *testing.T) {
	t.Parallel()

	testType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"test": tftypes.String,
		},
	}

	testValue := tftypes.NewValue(testType, map[string]tftypes.Value{
		"test": tftypes.NewValue(tftypes.String, "test-value"),
	})

	testDynamicValue, err := tfprotov6.NewDynamicValue(testType, testValue)

	if err != nil {
		t.Fatalf("unexpected error calling tfprotov6.NewDynamicValue(): %s", err)
	}

	testSchema := tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"test": {
				Required: true,
				Type:     types.StringType,
			},
		},
	}

	type testCase struct {
		server           *Server
		request          *tfprotov6.ConfigureProviderRequest
		expectedError    error
		expectedResponse *tfprotov6.ConfigureProviderResponse
	}

	testCases := map[string]testCase{
		"nil": {
			server: &Server{
				FrameworkServer: fwserver.Server{
					Provider: &testprovider.Provider{},
				},
			},
			request:          nil,
			expectedResponse: &tfprotov6.ConfigureProviderResponse{},
		},
		"no-schema": {
			server: &Server{
				FrameworkServer: fwserver.Server{
					Provider: &testprovider.Provider{
						GetSchemaMethod: func(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
							return tfsdk.Schema{}, nil
						},
						ConfigureMethod: func(ctx context.Context, req tfsdk.ConfigureProviderRequest, resp *tfsdk.ConfigureProviderResponse) {
							// Intentially empty, test is passing if it makes it this far
						},
					},
				},
			},
			request:          &tfprotov6.ConfigureProviderRequest{},
			expectedResponse: &tfprotov6.ConfigureProviderResponse{},
		},
		"request-config": {
			server: &Server{
				FrameworkServer: fwserver.Server{
					Provider: &testprovider.Provider{
						GetSchemaMethod: func(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
							return testSchema, nil
						},
						ConfigureMethod: func(ctx context.Context, req tfsdk.ConfigureProviderRequest, resp *tfsdk.ConfigureProviderResponse) {
							var got types.String

							resp.Diagnostics.Append(req.Config.GetAttribute(ctx, tftypes.NewAttributePath().WithAttributeName("test"), &got)...)

							if resp.Diagnostics.HasError() {
								return
							}

							if got.Value != "test-value" {
								resp.Diagnostics.AddError("Incorrect req.Config", "expected test-value, got "+got.Value)
							}
						},
					},
				},
			},
			request: &tfprotov6.ConfigureProviderRequest{
				Config: &testDynamicValue,
			},
			expectedResponse: &tfprotov6.ConfigureProviderResponse{},
		},
		"request-terraformversion": {
			server: &Server{
				FrameworkServer: fwserver.Server{
					Provider: &testprovider.Provider{
						GetSchemaMethod: func(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
							return tfsdk.Schema{}, nil
						},
						ConfigureMethod: func(ctx context.Context, req tfsdk.ConfigureProviderRequest, resp *tfsdk.ConfigureProviderResponse) {
							if req.TerraformVersion != "1.0.0" {
								resp.Diagnostics.AddError("Incorrect req.TerraformVersion", "expected 1.0.0, got "+req.TerraformVersion)
							}
						},
					},
				},
			},
			request: &tfprotov6.ConfigureProviderRequest{
				TerraformVersion: "1.0.0",
			},
			expectedResponse: &tfprotov6.ConfigureProviderResponse{},
		},
		"response-diagnostics": {
			server: &Server{
				FrameworkServer: fwserver.Server{
					Provider: &testprovider.Provider{
						GetSchemaMethod: func(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
							return tfsdk.Schema{}, nil
						},
						ConfigureMethod: func(ctx context.Context, req tfsdk.ConfigureProviderRequest, resp *tfsdk.ConfigureProviderResponse) {
							resp.Diagnostics.AddWarning("warning summary", "warning detail")
							resp.Diagnostics.AddError("error summary", "error detail")
						},
					},
				},
			},
			request: &tfprotov6.ConfigureProviderRequest{},
			expectedResponse: &tfprotov6.ConfigureProviderResponse{
				Diagnostics: []*tfprotov6.Diagnostic{
					{
						Severity: tfprotov6.DiagnosticSeverityWarning,
						Summary:  "warning summary",
						Detail:   "warning detail",
					},
					{
						Severity: tfprotov6.DiagnosticSeverityError,
						Summary:  "error summary",
						Detail:   "error detail",
					},
				},
			},
		},
	}

	for name, testCase := range testCases {
		name, testCase := name, testCase

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := testCase.server.ConfigureProvider(context.Background(), testCase.request)

			if diff := cmp.Diff(testCase.expectedError, err); diff != "" {
				t.Errorf("unexpected error difference: %s", diff)
			}

			if diff := cmp.Diff(testCase.expectedResponse, got); diff != "" {
				t.Errorf("unexpected response difference: %s", diff)
			}
		})
	}
}
