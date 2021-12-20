package rinex

/* func TestMeteoFile_Compress(t *testing.T) {
	assert := assert.New(t)
	tempDir := t.TempDir()

	// Rnx3
	rnxFilePath, err := copyToTempDir("testdata/white/DIEP00DEU_R_20202941900_01H_10S_MM.rnx", tempDir)
	if err != nil {
		t.Fatalf("Could not copy to temp dir: %v", err)
	}
	rnx3Fil, err := NewMeteoFile(rnxFilePath)
	assert.NoError(err)
	err = rnx3Fil.Compress()
	if err != nil {
		t.Fatalf("Could not Hatanaka compress file %s: %v", rnxFilePath, err)
	}
	assert.Equal(filepath.Join(tempDir, "DIEP00DEU_R_20202941900_01H_10S_MM.rnx.gz"), rnx3Fil.Path, "crx.gz file")
} */
