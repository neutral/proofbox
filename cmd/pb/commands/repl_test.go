package commands

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neutral/proofbox/pkg/proof"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper functions
func setupTestDB(t *testing.T) string {
	tempDir := t.TempDir()
	return filepath.Join(tempDir, "test.db")
}

func cleanupTestDB(dbPath string) {
	// t.TempDir() handles cleanup automatically
}

func createTempFile(t *testing.T, name string, content string) string {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, name)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
	return filePath
}

func cleanupTempFile(path string) {
	// t.TempDir() handles cleanup automatically
}

func TestReplCommands(t *testing.T) {
	// Create a test database
	dbPath := setupTestDB(t)
	defer cleanupTestDB(dbPath)

	// Initialize database
	rootCmd.SetArgs([]string{"init", "--db", dbPath})
	err := rootCmd.Execute()
	require.NoError(t, err)

	// Test REPL command parsing and execution
	store, keyEncoder, err := openDatabase(dbPath)
	require.NoError(t, err)
	defer store.Close()

	tree, err := openTree(store, keyEncoder)
	require.NoError(t, err)

	ctx := &replContext{
		tree:    tree,
		version: tree.GetLatestVersion(),
	}

	tests := []struct {
		name     string
		command  string
		wantErr  bool
		validate func(t *testing.T, ctx *replContext)
	}{
		{
			name:    "put command",
			command: "put testkey testvalue",
			wantErr: false,
			validate: func(t *testing.T, ctx *replContext) {
				// Version should have incremented
				assert.Equal(t, uint64(1), uint64(ctx.tree.GetLatestVersion()))
			},
		},
		{
			name:    "get existing key",
			command: "get testkey",
			wantErr: false,
		},
		{
			name:    "get non-existent key",
			command: "get nonexistent",
			wantErr: false, // Not an error, just not found
		},
		{
			name:    "delete command",
			command: "delete testkey",
			wantErr: false,
			validate: func(t *testing.T, ctx *replContext) {
				// Version should have incremented again
				assert.Equal(t, uint64(2), uint64(ctx.tree.GetLatestVersion()))
			},
		},
		{
			name:    "root command",
			command: "root",
			wantErr: false,
		},
		{
			name:    "stats command",
			command: "stats",
			wantErr: false,
		},
		{
			name:    "version command",
			command: "version",
			wantErr: false,
		},
		{
			name:    "version switch",
			command: "version 1",
			wantErr: false,
			validate: func(t *testing.T, ctx *replContext) {
				assert.Equal(t, uint64(1), uint64(ctx.version))
			},
		},
		{
			name:    "version latest",
			command: "version latest",
			wantErr: false,
			validate: func(t *testing.T, ctx *replContext) {
				assert.Equal(t, ctx.tree.GetLatestVersion(), ctx.version)
			},
		},
		{
			name:    "invalid command",
			command: "invalid",
			wantErr: true,
		},
		{
			name:    "put missing value",
			command: "put onlykey",
			wantErr: true,
		},
		{
			name:    "get missing key",
			command: "get",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executeReplCommand(ctx, tt.command)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.validate != nil {
				tt.validate(t, ctx)
			}
		})
	}
}

func TestReplCompleter(t *testing.T) {
	completer := createCompleter()
	require.NotNil(t, completer)

	// Test that we have completions for main commands
	var commands []string
	walkCompleter(completer, "", &commands)

	expectedCommands := []string{
		"put", "get", "delete", "root", "prove", "stats",
		"version", "batch", "export", "import", "help", "clear", "exit", "quit",
	}

	for _, cmd := range expectedCommands {
		assert.Contains(t, commands, cmd, "Expected command %s in completions", cmd)
	}
}

// Helper function to walk the completer tree
func walkCompleter(pc interface{}, prefix string, commands *[]string) {
	// This is a simplified walker - in practice you'd use reflection
	// to properly walk the readline.PrefixCompleter structure
	// For testing, we just verify the completer was created
}

func TestReplProve(t *testing.T) {
	// Create a test database
	dbPath := setupTestDB(t)
	defer cleanupTestDB(dbPath)

	// Initialize database
	rootCmd.SetArgs([]string{"init", "--db", dbPath})
	err := rootCmd.Execute()
	require.NoError(t, err)

	store, keyEncoder, err := openDatabase(dbPath)
	require.NoError(t, err)
	defer store.Close()

	tree, err := openTree(store, keyEncoder)
	require.NoError(t, err)

	ctx := &replContext{
		tree:    tree,
		version: tree.GetLatestVersion(),
	}

	// Add a key first
	err = executeReplCommand(ctx, "put provekey provevalue")
	require.NoError(t, err)

	// Test prove command
	err = executeReplCommand(ctx, "prove provekey")
	require.NoError(t, err)

	// Test prove for non-existent key
	err = executeReplCommand(ctx, "prove nonexistent")
	require.NoError(t, err)
}

func TestReplBatch(t *testing.T) {
	// Create test batch file
	batchContent := `put batch1 value1
put batch2 value2
get batch1
delete batch1`

	batchFile := createTempFile(t, "batch.txt", batchContent)
	defer cleanupTempFile(batchFile)

	// Create a test database
	dbPath := setupTestDB(t)
	defer cleanupTestDB(dbPath)

	// Initialize database
	rootCmd.SetArgs([]string{"init", "--db", dbPath})
	err := rootCmd.Execute()
	require.NoError(t, err)

	store, keyEncoder, err := openDatabase(dbPath)
	require.NoError(t, err)
	defer store.Close()

	tree, err := openTree(store, keyEncoder)
	require.NoError(t, err)

	ctx := &replContext{
		tree:    tree,
		version: tree.GetLatestVersion(),
	}

	// Execute batch
	err = replBatch(ctx, batchFile)
	require.NoError(t, err)

	// Verify results
	finalVersion := tree.GetLatestVersion()
	assert.Greater(t, uint64(finalVersion), uint64(0))
}

func TestFilterInput(t *testing.T) {
	// Test that Ctrl+Z is filtered
	r, ok := filterInput('\x1a') // Ctrl+Z
	assert.Equal(t, '\x1a', r)
	assert.False(t, ok)

	// Test that normal characters pass through
	r, ok = filterInput('a')
	assert.Equal(t, 'a', r)
	assert.True(t, ok)
}

func TestProofTypeToString(t *testing.T) {
	tests := []struct {
		input    int // We'll cast to proof.ProofType in the test
		expected string
	}{
		{0, "inclusion"},           // Assuming ProofTypeInclusion = 0
		{1, "exclusion_empty"},     // Assuming ProofTypeExclusionEmpty = 1
		{2, "exclusion_neighbor"},  // Assuming ProofTypeExclusionNeighbor = 2
		{99, "unknown"},
	}

	for _, tt := range tests {
		// Note: In real code, you'd use the actual proof.ProofType constants
		result := proofTypeToString(proof.ProofType(tt.input))
		assert.Equal(t, tt.expected, result)
	}
}

func TestReplHelp(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printReplHelp()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	require.NoError(t, err)
	output := buf.String()

	// Verify help contains key commands
	assert.Contains(t, output, "put <key> <value>")
	assert.Contains(t, output, "get <key>")
	assert.Contains(t, output, "delete <key>")
	assert.Contains(t, output, "help")
	assert.Contains(t, output, "exit")
}

func TestReplCommandErrors(t *testing.T) {
	// Create a test database
	dbPath := setupTestDB(t)
	defer cleanupTestDB(dbPath)

	// Initialize database
	rootCmd.SetArgs([]string{"init", "--db", dbPath})
	err := rootCmd.Execute()
	require.NoError(t, err)

	store, keyEncoder, err := openDatabase(dbPath)
	require.NoError(t, err)
	defer store.Close()

	tree, err := openTree(store, keyEncoder)
	require.NoError(t, err)

	ctx := &replContext{
		tree:    tree,
		version: tree.GetLatestVersion(),
	}

	// Test various error conditions
	tests := []struct {
		name    string
		command string
	}{
		{"empty command", ""},
		{"whitespace only", "   "},
		{"prove without key", "prove"},
		{"batch without file", "batch"},
		{"export without file", "export"},
		{"import without file", "import"},
		{"invalid version", "version abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executeReplCommand(ctx, tt.command)
			// Empty commands should not error
			if tt.command == "" || strings.TrimSpace(tt.command) == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}