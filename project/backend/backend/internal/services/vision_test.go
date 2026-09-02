package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVisionService(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	t.Run("creates service with default config", func(t *testing.T) {
		service := NewVisionService(nil, logger, nil)
		assert.NotNil(t, service)
		assert.Equal(t, "./storage/videos", service.storagePath)
		assert.Equal(t, int64(1024), service.maxStorageMB)
		assert.True(t, service.compressionOn)
		// Verify CV components are initialized
		assert.NotNil(t, service.opticalFlow)
		assert.NotNil(t, service.boilCalculator)
		assert.NotNil(t, service.blobDetector)
		assert.NotNil(t, service.surfaceAnalyzer)
	})

	t.Run("creates service with custom config", func(t *testing.T) {
		config := &VisionServiceConfig{
			StoragePath:   "./custom/path",
			MaxStorageMB:  2048,
			CompressionOn: false,
		}
		service := NewVisionService(nil, logger, config)
		assert.NotNil(t, service)
		assert.Equal(t, "./custom/path", service.storagePath)
		assert.Equal(t, int64(2048), service.maxStorageMB)
		assert.False(t, service.compressionOn)
	})
}

func TestVisionService_UploadVideo(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	// Create temp directory for tests
	tempDir := t.TempDir()
	config := &VisionServiceConfig{
		StoragePath:   tempDir,
		MaxStorageMB:  100,
		CompressionOn: false,
	}
	service := NewVisionService(nil, logger, config)

	t.Run("uploads video successfully", func(t *testing.T) {
		ctx := context.Background()
		deviceID := "test-device-001"
		filename := "test_video.mp4"
		data := []byte("fake video data for testing")

		result, err := service.UploadVideo(ctx, deviceID, filename, data)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, filename, result.Filename)
		assert.Equal(t, int64(len(data)), result.FileSize)
		assert.NotEmpty(t, result.Checksum)
		assert.NotEmpty(t, result.FilePath)

		// Verify file exists
		_, err = os.Stat(result.FilePath)
		assert.NoError(t, err)
	})

	t.Run("creates device directory", func(t *testing.T) {
		ctx := context.Background()
		deviceID := "new-device-002"
		filename := "video.mp4"
		data := []byte("test data")

		result, err := service.UploadVideo(ctx, deviceID, filename, data)
		require.NoError(t, err)

		// Verify device directory was created
		deviceDir := filepath.Join(tempDir, deviceID)
		info, err := os.Stat(deviceDir)
		assert.NoError(t, err)
		assert.True(t, info.IsDir())
		assert.Contains(t, result.FilePath, deviceID)
	})
}

func TestVisionService_UploadVideoChunk(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	tempDir := t.TempDir()
	config := &VisionServiceConfig{
		StoragePath:   tempDir,
		MaxStorageMB:  100,
		CompressionOn: false,
	}
	service := NewVisionService(nil, logger, config)

	t.Run("uploads chunks and assembles file", func(t *testing.T) {
		ctx := context.Background()
		deviceID := "chunk-device"
		filename := "chunked_video.mp4"

		chunk1 := []byte("chunk1 data ")
		chunk2 := []byte("chunk2 data ")
		chunk3 := []byte("chunk3 data")

		// Upload first chunk
		result1, err := service.UploadVideoChunk(ctx, deviceID, filename, 0, 3, chunk1)
		require.NoError(t, err)
		assert.Equal(t, filename, result1.Filename)

		// Upload second chunk
		result2, err := service.UploadVideoChunk(ctx, deviceID, filename, 1, 3, chunk2)
		require.NoError(t, err)
		assert.Equal(t, filename, result2.Filename)

		// Upload final chunk - should trigger assembly
		result3, err := service.UploadVideoChunk(ctx, deviceID, filename, 2, 3, chunk3)
		require.NoError(t, err)
		assert.NotEmpty(t, result3.FilePath)
		assert.NotEmpty(t, result3.Checksum)

		// Verify assembled file
		assembledData, err := os.ReadFile(result3.FilePath)
		require.NoError(t, err)
		expectedData := append(append(chunk1, chunk2...), chunk3...)
		assert.Equal(t, expectedData, assembledData)
	})
}

func TestVisionService_CompressVideo(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	tempDir := t.TempDir()
	config := &VisionServiceConfig{
		StoragePath:   tempDir,
		MaxStorageMB:  100,
		CompressionOn: true,
	}
	service := NewVisionService(nil, logger, config)

	t.Run("compresses compressible data", func(t *testing.T) {
		// Create highly compressible data
		data := make([]byte, 10000)
		for i := range data {
			data[i] = 'A' // Repetitive data compresses well
		}

		compressed, err := service.compressVideo(data)
		require.NoError(t, err)
		assert.Less(t, len(compressed), len(data))
	})

	t.Run("decompress returns original data", func(t *testing.T) {
		data := make([]byte, 1000)
		for i := range data {
			data[i] = byte(i % 256)
		}

		compressed, err := service.compressVideo(data)
		require.NoError(t, err)

		// Only decompress if it was actually compressed
		if len(compressed) < len(data) {
			decompressed, err := service.DecompressVideo(compressed)
			require.NoError(t, err)
			assert.Equal(t, data, decompressed)
		}
	})
}

func TestVisionService_AnalyzeImage_WithRealCV(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	tempDir := t.TempDir()
	config := &VisionServiceConfig{
		StoragePath:   tempDir,
		MaxStorageMB:  100,
		CompressionOn: false,
	}
	service := NewVisionService(nil, logger, config)

	t.Run("analyzes image using real CV algorithms", func(t *testing.T) {
		ctx := context.Background()
		deviceID := "test-device"
		imagePath := "/path/to/test/image.jpg"

		// Reset analyzers to ensure clean state
		service.ResetAnalyzers()

		analysis, err := service.AnalyzeImage(ctx, deviceID, imagePath, nil)
		require.NoError(t, err)
		assert.NotNil(t, analysis)
		assert.Equal(t, deviceID, analysis.DeviceID)
		assert.Equal(t, imagePath, analysis.ImagePath)
		assert.GreaterOrEqual(t, analysis.FeedingActivityScore, 0.0)
		assert.LessOrEqual(t, analysis.FeedingActivityScore, 1.0)
		assert.GreaterOrEqual(t, analysis.SatietyLevel, 0.0)
		assert.LessOrEqual(t, analysis.SatietyLevel, 1.0)
		assert.Equal(t, "cv_algorithms_v1.0", analysis.AnalysisModel)
	})

	t.Run("includes video clip ID when provided", func(t *testing.T) {
		ctx := context.Background()
		deviceID := "test-device"
		imagePath := "/path/to/image.jpg"
		videoClipID := uint(123)

		service.ResetAnalyzers()

		analysis, err := service.AnalyzeImage(ctx, deviceID, imagePath, &videoClipID)
		require.NoError(t, err)
		assert.NotNil(t, analysis.VideoClipID)
		assert.Equal(t, videoClipID, *analysis.VideoClipID)
	})

	t.Run("produces consistent results for same image path", func(t *testing.T) {
		ctx := context.Background()
		deviceID := "test-device"
		imagePath := "/consistent/test/path.jpg"

		// Reset and analyze twice
		service.ResetAnalyzers()
		analysis1, err := service.AnalyzeImage(ctx, deviceID, imagePath, nil)
		require.NoError(t, err)

		service.ResetAnalyzers()
		analysis2, err := service.AnalyzeImage(ctx, deviceID, imagePath, nil)
		require.NoError(t, err)

		// Results should be consistent for same input
		assert.Equal(t, analysis1.FeedingActivityScore, analysis2.FeedingActivityScore)
	})
}

func TestVisionService_AnalyzeBoilIndex_WithRealCV(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	tempDir := t.TempDir()
	config := &VisionServiceConfig{
		StoragePath:   tempDir,
		MaxStorageMB:  100,
		CompressionOn: false,
	}
	service := NewVisionService(nil, logger, config)

	t.Run("analyzes boil index using real CV algorithms", func(t *testing.T) {
		ctx := context.Background()
		deviceID := "test-device"
		imagePath := "/path/to/feeding/image.jpg"

		service.ResetAnalyzers()

		analysis, err := service.AnalyzeBoilIndex(ctx, deviceID, nil, imagePath)
		require.NoError(t, err)
		assert.NotNil(t, analysis)
		assert.Equal(t, deviceID, analysis.DeviceID)
		assert.GreaterOrEqual(t, analysis.PreFeedBoilIndex, 0.0)
		assert.LessOrEqual(t, analysis.PreFeedBoilIndex, 1.0)
		assert.GreaterOrEqual(t, analysis.ActiveFeedBoilIndex, 0.0)
		assert.LessOrEqual(t, analysis.ActiveFeedBoilIndex, 1.0)
		assert.GreaterOrEqual(t, analysis.PostFeedBoilIndex, 0.0)
		assert.LessOrEqual(t, analysis.PostFeedBoilIndex, 1.0)
		assert.Equal(t, 0.4, analysis.SatietyThreshold)
		assert.Equal(t, "boil_index_cv_v1.2", analysis.AlgorithmVersion)
	})

	t.Run("includes feeding event ID when provided", func(t *testing.T) {
		ctx := context.Background()
		deviceID := "test-device"
		imagePath := "/path/to/image.jpg"
		feedingEventID := uint(456)

		service.ResetAnalyzers()

		analysis, err := service.AnalyzeBoilIndex(ctx, deviceID, &feedingEventID, imagePath)
		require.NoError(t, err)
		assert.NotNil(t, analysis.FeedingEventID)
		assert.Equal(t, feedingEventID, *analysis.FeedingEventID)
	})
}

func TestVisionService_OpticalFlowAnalysis(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	service := NewVisionService(nil, logger, nil)

	t.Run("performs optical flow analysis", func(t *testing.T) {
		ctx := context.Background()
		imagePath := "/test/optical/flow.jpg"

		service.ResetAnalyzers()

		// First call initializes the analyzer
		result1, err := service.AnalyzeOpticalFlow(ctx, imagePath)
		require.NoError(t, err)
		assert.NotNil(t, result1)

		// Second call with different image should produce flow
		result2, err := service.AnalyzeOpticalFlow(ctx, "/test/optical/flow2.jpg")
		require.NoError(t, err)
		assert.NotNil(t, result2)
		assert.GreaterOrEqual(t, result2.ActivityLevel, 0.0)
		assert.LessOrEqual(t, result2.ActivityLevel, 1.0)
	})
}

func TestVisionService_DetectPellets(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	service := NewVisionService(nil, logger, nil)

	t.Run("detects pellets using blob detection", func(t *testing.T) {
		ctx := context.Background()
		deviceID := "test-device"
		imagePath := "/test/pellet/image.jpg"

		result, err := service.DetectPellets(ctx, deviceID, imagePath)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, deviceID, result.DeviceID)
		assert.Equal(t, imagePath, result.ImagePath)
		assert.GreaterOrEqual(t, result.PelletCount, 0)
		assert.GreaterOrEqual(t, result.Confidence, 0.0)
		assert.LessOrEqual(t, result.Confidence, 1.0)
		assert.Greater(t, result.ProcessingTimeMs, 0)
	})
}

func TestVisionService_CalculateSatietyFromActivity(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	service := NewVisionService(nil, logger, nil)

	tests := []struct {
		name          string
		activityLevel float64
		expectedMin   float64
		expectedMax   float64
	}{
		{"very high activity - very hungry", 0.9, 0.0, 0.2},
		{"high activity - hungry", 0.7, 0.2, 0.4},
		{"moderate activity - moderate satiety", 0.5, 0.4, 0.6},
		{"low activity - satisfied", 0.3, 0.6, 0.8},
		{"very low activity - full", 0.1, 0.8, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			satiety := service.calculateSatietyFromActivity(tt.activityLevel)
			assert.GreaterOrEqual(t, satiety, tt.expectedMin)
			assert.LessOrEqual(t, satiety, tt.expectedMax)
		})
	}
}

func TestVisionService_StorageUsage(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	t.Run("returns error when repo is nil", func(t *testing.T) {
		service := NewVisionService(nil, logger, nil)
		ctx := context.Background()

		usage, err := service.GetStorageUsage(ctx, "test-device")
		assert.Error(t, err)
		assert.Nil(t, usage)
		assert.Contains(t, err.Error(), "repository not initialized")
	})
}

func TestVisionService_VideoClipOperations(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	service := NewVisionService(nil, logger, nil)
	ctx := context.Background()

	t.Run("GetVideoClip returns error when repo is nil", func(t *testing.T) {
		clip, err := service.GetVideoClip(ctx, 1)
		assert.Error(t, err)
		assert.Nil(t, clip)
	})

	t.Run("GetVideoClipsByDevice returns error when repo is nil", func(t *testing.T) {
		clips, err := service.GetVideoClipsByDevice(ctx, "device", 10)
		assert.Error(t, err)
		assert.Nil(t, clips)
	})

	t.Run("GetVideoClipsByFeedingEvent returns error when repo is nil", func(t *testing.T) {
		clips, err := service.GetVideoClipsByFeedingEvent(ctx, 1)
		assert.Error(t, err)
		assert.Nil(t, clips)
	})

	t.Run("DeleteVideoClip returns error when repo is nil", func(t *testing.T) {
		err := service.DeleteVideoClip(ctx, 1)
		assert.Error(t, err)
	})
}

func TestVisionService_AnalysisOperations(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	service := NewVisionService(nil, logger, nil)
	ctx := context.Background()

	t.Run("GetImageAnalyses returns error when repo is nil", func(t *testing.T) {
		analyses, err := service.GetImageAnalyses(ctx, "device", 10)
		assert.Error(t, err)
		assert.Nil(t, analyses)
	})

	t.Run("GetBoilIndexAnalyses returns error when repo is nil", func(t *testing.T) {
		analyses, err := service.GetBoilIndexAnalyses(ctx, "device", 10)
		assert.Error(t, err)
		assert.Nil(t, analyses)
	})

	t.Run("GetVisionStats returns error when repo is nil", func(t *testing.T) {
		stats, err := service.GetVisionStats(ctx, "device", time.Now().AddDate(0, 0, -7), time.Now())
		assert.Error(t, err)
		assert.Nil(t, stats)
	})
}

// Property-based test for computer vision data processing (Property 25)
func TestProperty25_ComputerVisionDataAccuracy(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	tempDir := t.TempDir()
	config := &VisionServiceConfig{
		StoragePath:   tempDir,
		MaxStorageMB:  100,
		CompressionOn: false,
	}
	service := NewVisionService(nil, logger, config)
	ctx := context.Background()

	t.Run("Property: Image analysis with real CV produces valid bounded results", func(t *testing.T) {
		testPaths := []string{
			"/test/image1.jpg",
			"/test/image2.jpg",
			"/feeding/capture.png",
			"/device/snapshot.jpg",
		}

		for _, path := range testPaths {
			service.ResetAnalyzers()
			analysis, err := service.AnalyzeImage(ctx, "test-device", path, nil)
			require.NoError(t, err)

			// Property: All scores must be in [0, 1] range
			assert.GreaterOrEqual(t, analysis.FeedingActivityScore, 0.0,
				"FeedingActivityScore must be >= 0 for path %s", path)
			assert.LessOrEqual(t, analysis.FeedingActivityScore, 1.0,
				"FeedingActivityScore must be <= 1 for path %s", path)
			assert.GreaterOrEqual(t, analysis.SatietyLevel, 0.0,
				"SatietyLevel must be >= 0 for path %s", path)
			assert.LessOrEqual(t, analysis.SatietyLevel, 1.0,
				"SatietyLevel must be <= 1 for path %s", path)

			// Property: Pellet count must be non-negative
			assert.GreaterOrEqual(t, analysis.UneatePelletsCount, 0,
				"UneatePelletsCount must be >= 0 for path %s", path)

			// Property: FeedingActivity boolean must match score threshold
			if analysis.FeedingActivityScore > 0.3 {
				assert.True(t, analysis.FeedingActivity,
					"FeedingActivity should be true when score > 0.3 for path %s", path)
			} else {
				assert.False(t, analysis.FeedingActivity,
					"FeedingActivity should be false when score <= 0.3 for path %s", path)
			}
		}
	})

	t.Run("Property: Boil index analysis with real CV produces consistent results", func(t *testing.T) {
		testPaths := []string{
			"/feeding/pre.jpg",
			"/feeding/active.jpg",
			"/feeding/post.jpg",
		}

		for _, path := range testPaths {
			service.ResetAnalyzers()
			analysis, err := service.AnalyzeBoilIndex(ctx, "test-device", nil, path)
			require.NoError(t, err)

			// Property: All indices must be in [0, 1] range
			assert.GreaterOrEqual(t, analysis.PreFeedBoilIndex, 0.0,
				"PreFeedBoilIndex must be >= 0 for path %s", path)
			assert.LessOrEqual(t, analysis.PreFeedBoilIndex, 1.0,
				"PreFeedBoilIndex must be <= 1 for path %s", path)
			assert.GreaterOrEqual(t, analysis.ActiveFeedBoilIndex, 0.0,
				"ActiveFeedBoilIndex must be >= 0 for path %s", path)
			assert.LessOrEqual(t, analysis.ActiveFeedBoilIndex, 1.0,
				"ActiveFeedBoilIndex must be <= 1 for path %s", path)
			assert.GreaterOrEqual(t, analysis.PostFeedBoilIndex, 0.0,
				"PostFeedBoilIndex must be >= 0 for path %s", path)
			assert.LessOrEqual(t, analysis.PostFeedBoilIndex, 1.0,
				"PostFeedBoilIndex must be <= 1 for path %s", path)
			assert.GreaterOrEqual(t, analysis.FeedingEfficiency, 0.0,
				"FeedingEfficiency must be >= 0 for path %s", path)
			assert.LessOrEqual(t, analysis.FeedingEfficiency, 1.0,
				"FeedingEfficiency must be <= 1 for path %s", path)

			// Property: Satiety threshold must be 0.4
			assert.Equal(t, 0.4, analysis.SatietyThreshold,
				"SatietyThreshold must be 0.4 for path %s", path)
		}
	})

	t.Run("Property: Video upload preserves data integrity", func(t *testing.T) {
		testData := [][]byte{
			[]byte("small video data"),
			make([]byte, 1000),
			make([]byte, 10000),
		}

		for i, data := range testData {
			// Fill with pattern for verification
			for j := range data {
				data[j] = byte((i + j) % 256)
			}

			result, err := service.UploadVideo(ctx, "test-device", "test.mp4", data)
			require.NoError(t, err)

			// Property: Checksum must be non-empty
			assert.NotEmpty(t, result.Checksum)

			// Property: File size must match data size (no compression in this config)
			assert.Equal(t, int64(len(data)), result.FileSize)

			// Property: File must exist at reported path
			_, err = os.Stat(result.FilePath)
			assert.NoError(t, err)

			// Property: File content must match original
			savedData, err := os.ReadFile(result.FilePath)
			require.NoError(t, err)
			assert.Equal(t, data, savedData)
		}
	})

	t.Run("Property: Chunked upload assembles correctly", func(t *testing.T) {
		chunks := [][]byte{
			[]byte("chunk0_data_"),
			[]byte("chunk1_data_"),
			[]byte("chunk2_data"),
		}

		var lastResult *VideoUploadResult
		for i, chunk := range chunks {
			result, err := service.UploadVideoChunk(ctx, "chunk-device", "assembled.mp4", i, len(chunks), chunk)
			require.NoError(t, err)
			lastResult = result
		}

		// Property: Final result must have complete file info
		assert.NotEmpty(t, lastResult.FilePath)
		assert.NotEmpty(t, lastResult.Checksum)

		// Property: Assembled file must contain all chunks in order
		assembledData, err := os.ReadFile(lastResult.FilePath)
		require.NoError(t, err)

		expectedData := []byte{}
		for _, chunk := range chunks {
			expectedData = append(expectedData, chunk...)
		}
		assert.Equal(t, expectedData, assembledData)
	})

	t.Run("Property: Satiety calculation is monotonically decreasing with activity", func(t *testing.T) {
		activityLevels := []float64{0.1, 0.3, 0.5, 0.7, 0.9}
		var prevSatiety float64 = 1.0

		for _, activity := range activityLevels {
			satiety := service.calculateSatietyFromActivity(activity)

			// Property: Higher activity should result in lower or equal satiety
			assert.LessOrEqual(t, satiety, prevSatiety,
				"Satiety should decrease as activity increases")
			prevSatiety = satiety
		}
	})

	t.Run("Property: Optical flow analysis produces valid results", func(t *testing.T) {
		service.ResetAnalyzers()

		// Initialize with first frame
		_, err := service.AnalyzeOpticalFlow(ctx, "/frame1.jpg")
		require.NoError(t, err)

		// Analyze second frame
		result, err := service.AnalyzeOpticalFlow(ctx, "/frame2.jpg")
		require.NoError(t, err)

		// Property: Activity level must be in [0, 1] range
		assert.GreaterOrEqual(t, result.ActivityLevel, 0.0)
		assert.LessOrEqual(t, result.ActivityLevel, 1.0)

		// Property: Confidence must be in [0, 1] range
		assert.GreaterOrEqual(t, result.Confidence, 0.0)
		assert.LessOrEqual(t, result.Confidence, 1.0)
	})

	t.Run("Property: Pellet detection produces valid results", func(t *testing.T) {
		testPaths := []string{
			"/pellet/test1.jpg",
			"/pellet/test2.jpg",
			"/pellet/test3.jpg",
		}

		for _, path := range testPaths {
			result, err := service.DetectPellets(ctx, "test-device", path)
			require.NoError(t, err)

			// Property: Pellet count must be non-negative
			assert.GreaterOrEqual(t, result.PelletCount, 0,
				"PelletCount must be >= 0 for path %s", path)

			// Property: Confidence must be in [0, 1] range
			assert.GreaterOrEqual(t, result.Confidence, 0.0,
				"Confidence must be >= 0 for path %s", path)
			assert.LessOrEqual(t, result.Confidence, 1.0,
				"Confidence must be <= 1 for path %s", path)

			// Property: Coverage must be in [0, 100] range
			assert.GreaterOrEqual(t, result.CoveragePercentage, 0.0,
				"CoveragePercentage must be >= 0 for path %s", path)
			assert.LessOrEqual(t, result.CoveragePercentage, 100.0,
				"CoveragePercentage must be <= 100 for path %s", path)

			// Property: PelletsDetected must match PelletCount
			if result.PelletCount > 0 {
				assert.True(t, result.PelletsDetected,
					"PelletsDetected should be true when PelletCount > 0 for path %s", path)
			} else {
				assert.False(t, result.PelletsDetected,
					"PelletsDetected should be false when PelletCount == 0 for path %s", path)
			}
		}
	})
}
