package operator

import (
	"testing"

	configv1 "github.com/openshift/api/config/v1"
)

// Unit Test: validateUserTag
func TestValidateUserTag(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid tag",
			key:       "valid-key",
			value:     "valid-value",
			expectErr: false,
		},
		{
			name:      "invalid key (special characters)",
			key:       "invalid*key",
			value:     "valid-value",
			expectErr: true,
			errMsg:    "key has invalid characters or length",
		},
		{
			name:      "invalid value (special characters)",
			key:       "valid-key",
			value:     "invalid*value",
			expectErr: true,
			errMsg:    "value has invalid characters or length",
		},
		{
			name:      "disallowed key 'Name'",
			key:       "Name",
			value:     "value",
			expectErr: true,
			errMsg:    "name key is not allowed for user defined tags",
		},
		{
			name:      "kubernetes.io namespace in key",
			key:       "kubernetes.io/namespace",
			value:     "value",
			expectErr: true,
			errMsg:    "key is in the kubernetes.io namespace",
		},
		{
			name:      "openshift.io namespace in key",
			key:       "openshift.io/namespace",
			value:     "value",
			expectErr: true,
			errMsg:    "key is in the openshift.io namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUserTag(tt.key, tt.value)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected an error but got none")
				} else if !containsErrorMessage(err.Error(), tt.errMsg) {
					t.Errorf("expected error message to contain %q, but got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

// Helper function to check if an error message contains a substring
func containsErrorMessage(err, msg string) bool {
	return msg == "" || (err != "" && msg != "" && (len(err) >= len(msg) && err[:len(msg)] == msg))
}

// Unit Test: syncInfraTags
func TestSyncInfraTags(t *testing.T) {
	tests := []struct {
		name              string
		s3TagSet          map[string]string
		infraTagSet       map[string]string
		expectedUpdated   int
		expectedFinalTags map[string]string
	}{
		{
			name: "no new tags to update",
			s3TagSet: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			infraTagSet: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expectedUpdated: 0,
			expectedFinalTags: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "new tags added",
			s3TagSet: map[string]string{
				"key1": "value1",
			},
			infraTagSet: map[string]string{
				"key1": "value1",
				"key2": "new-value",
			},
			expectedUpdated: 1,
			expectedFinalTags: map[string]string{
				"key1": "value1",
				"key2": "new-value",
			},
		},
		{
			name: "existing tags updated",
			s3TagSet: map[string]string{
				"key1": "value1",
			},
			infraTagSet: map[string]string{
				"key1": "updated-value",
			},
			expectedUpdated: 1,
			expectedFinalTags: map[string]string{
				"key1": "updated-value",
			},
		},
		{
			name: "multiple tags added and updated",
			s3TagSet: map[string]string{
				"key1": "value1",
			},
			infraTagSet: map[string]string{
				"key1": "updated-value",
				"key2": "new-value",
			},
			expectedUpdated: 2,
			expectedFinalTags: map[string]string{
				"key1": "updated-value",
				"key2": "new-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedCount := syncInfraTags(tt.s3TagSet, tt.infraTagSet)
			if updatedCount != tt.expectedUpdated {
				t.Errorf("expected %d updates, but got %d", tt.expectedUpdated, updatedCount)
			}

			for key, value := range tt.expectedFinalTags {
				if tt.s3TagSet[key] != value {
					t.Errorf("expected final tag %q to be %q, but got %q", key, value, tt.s3TagSet[key])
				}
			}
		})
	}
}

// Unit Test: filterPlatformStatusTags
func TestFilterPlatformStatusTags(t *testing.T) {
	tests := []struct {
		name         string
		infra        *configv1.Infrastructure
		expectedTags map[string]string
	}{
		{
			name: "valid tags",
			infra: &configv1.Infrastructure{
				Status: configv1.InfrastructureStatus{
					PlatformStatus: &configv1.PlatformStatus{
						AWS: &configv1.AWSPlatformStatus{
							ResourceTags: []configv1.AWSResourceTag{
								{Key: "valid-key", Value: "valid-value"},
								{Key: "another-valid-key", Value: "another-valid-value"},
							},
						},
					},
				},
			},
			expectedTags: map[string]string{
				"valid-key":         "valid-value",
				"another-valid-key": "another-valid-value",
			},
		},
		{
			name: "invalid tags are filtered out",
			infra: &configv1.Infrastructure{
				Status: configv1.InfrastructureStatus{
					PlatformStatus: &configv1.PlatformStatus{
						AWS: &configv1.AWSPlatformStatus{
							ResourceTags: []configv1.AWSResourceTag{
								{Key: "invalid*key", Value: "valid-value"},
								{Key: "valid-key", Value: "valid-value"},
							},
						},
					},
				},
			},
			expectedTags: map[string]string{
				"valid-key": "valid-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := filterPlatformStatusTags(tt.infra)

			if len(tags) != len(tt.expectedTags) {
				t.Fatalf("expected %d tags, but got %d", len(tt.expectedTags), len(tags))
			}

			for key, value := range tt.expectedTags {
				if tags[key] != value {
					t.Errorf("expected tag %q to be %q, but got %q", key, value, tags[key])
				}
			}
		})
	}
}
