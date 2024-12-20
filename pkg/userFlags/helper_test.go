package userflags

// func TestGenerateFilePathList(t *testing.T) {
// 	tests := []struct {
// 		name           string
// 		fileName       string
// 		fp             string
// 		expectedErrMsg string
// 		expected       []string
// 		expectedErr    bool
// 	}{
// 		{
// 			name:           "Both fileName and fp are empty",
// 			fileName:       "",
// 			fp:             "",
// 			expected:       nil,
// 			expectedErr:    true,
// 			expectedErrMsg: "to send api request(s), please provide a valid file name with \n'-f fileName' flag or  \n'-fp file/path/' ",
// 		},
// 		{
// 			name:        "Only fp is provided",
// 			fileName:    "",
// 			fp:          "path/to/directory",
// 			expected:    []string{"path/to/directory"},
// 			expectedErr: false,
// 		},
// 		{
// 			name:        "Only fileName is provided and matches files",
// 			fileName:    "validFile",
// 			fp:          "",
// 			expected:    []string{"path/to/validFile.yaml", "path/to/validFile.yml"},
// 			expectedErr: false,
// 		},
// 		{
// 			name:           "Only fileName is provided but no files match",
// 			fileName:       "emptyFile",
// 			fp:             "",
// 			expected:       nil,
// 			expectedErr:    true,
// 			expectedErrMsg: "to send api request(s), please provide a valid file name with \n'-f fileName' flag or  \n'-fp file/path/' ",
// 		},
// 		{
// 			name:           "Only fileName is provided but an error occurs",
// 			fileName:       "errorFile",
// 			fp:             "",
// 			expected:       nil,
// 			expectedErr:    true,
// 			expectedErrMsg: "to send api request(s), please provide a valid file name with \n'-f fileName' flag or  \n'-fp file/path/' ",
// 		},
// 		{
// 			name:     "Both fileName and fp are provided",
// 			fileName: "validFile",
// 			fp:       "path/to/directory",
// 			expected: []string{
// 				"path/to/directory",
// 				"path/to/validFile.yaml",
// 				"path/to/validFile.yml",
// 			},
// 			expectedErr: false,
// 		},
// 		{
// 			name:        "Both fileName and fp are provided but no matches for fileName",
// 			fileName:    "emptyFile",
// 			fp:          "path/to/directory",
// 			expected:    []string{"path/to/directory"},
// 			expectedErr: false,
// 		},
// 		{
// 			name:        "Both fileName and fp are provided but error for fileName",
// 			fileName:    "errorFile",
// 			fp:          "path/to/directory",
// 			expected:    []string{"path/to/directory"},
// 			expectedErr: false,
// 		},
// 	}
//
// 	for _, tc := range tests {
// 		t.Run(tc.name, func(t *testing.T) {
// 			result, err := GenerateFilePathList(tc.fileName, tc.fp)
//
// 			// Validate error
// 			if tc.expectedErr {
// 				if err == nil {
// 					t.Errorf("Test %q failed: expected an error but got none", tc.name)
// 				} else if err.Error() != tc.expectedErrMsg {
// 					t.Errorf(
// 						"Test %q failed: expected error message %q, got %q",
// 						tc.name,
// 						tc.expectedErrMsg,
// 						err.Error(),
// 					)
// 				}
// 			} else {
// 				if err != nil {
// 					t.Errorf("Test %q failed: did not expect an error but got %v", tc.name, err)
// 				}
// 			}
//
// 			// Validate result
// 			if len(result) != len(tc.expected) {
// 				t.Errorf(
// 					"Test %q failed: expected length %d, got %d",
// 					tc.name,
// 					len(tc.expected),
// 					len(result),
// 				)
// 			} else {
// 				for i := range result {
// 					if result[i] != tc.expected[i] {
// 						t.Errorf(
// 							"Test %q failed: expected %v, got %v",
// 							tc.name,
// 							tc.expected,
// 							result,
// 						)
// 						break
// 					}
// 				}
// 			}
// 		})
// 	}
// }
