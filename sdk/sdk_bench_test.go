// Copyright (C) 2024 Akave
// See LICENSE for copying information.

package sdk_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"akave.ai/akavesdk/private/memory"
	"akave.ai/akavesdk/sdk"
)

func BenchmarkUploadDownload(b *testing.B) {
	var maxConcurrencyValues = []int{5, 10, 15, 20, 25, 30, 35, 40, 45, 50}
	var files = []*bytes.Buffer{
		generateFile(b, 2024, 10*memory.MB.ToInt()),
		generateFile(b, 2024, 100*memory.MB.ToInt()),
		generateFile(b, 2024, 256*memory.MB.ToInt()),
		generateFile(b, 2024, 512*memory.MB.ToInt()),
		generateFile(b, 2024, 1024*memory.MB.ToInt()),
	}
	for _, maxConcurrency := range maxConcurrencyValues {
		b.Run(fmt.Sprintf("MaxConcurrency_%d", maxConcurrency), func(b *testing.B) {
			for _, file := range files {
				b.Run(fmt.Sprintf("FileSize %s", memory.FormatBytes(file.Len())), func(b *testing.B) {
					b.Run("Standalone connection", func(b *testing.B) {
						akave, err := sdk.New(PickNodeRPCAddress(b), maxConcurrency, chunkSegmentSize.ToInt64(), false)
						require.NoError(b, err)

						b.Cleanup(func() {
							err = akave.Close()
							require.NoError(b, err)
						})

						doUploadDownload(b, akave, *file)
					})

					b.Run("With Pool", func(b *testing.B) {
						akave, err := sdk.New(PickNodeRPCAddress(b), maxConcurrency, chunkSegmentSize.ToInt64(), true)
						require.NoError(b, err)

						b.Cleanup(func() {
							err = akave.Close()
							require.NoError(b, err)
						})

						doUploadDownload(b, akave, *file)
					})
				})
			}
		})
	}
}

func doUploadDownload(b testing.TB, sdk *sdk.SDK, file bytes.Buffer) {
	bucketName := randomBucketName(b, 10)
	expected := file.Bytes()

	// create bucket
	_, err := sdk.CreateBucket(context.Background(), bucketName)
	require.NoError(b, err)

	fileUpload, err := sdk.CreateFileUpload(context.Background(), bucketName, "example.txt", int64(file.Len()), &file)
	require.NoError(b, err)

	err = sdk.Upload(context.Background(), fileUpload)
	require.NoError(b, err)

	var downloaded bytes.Buffer
	fileDownload, err := sdk.CreateFileDownloadV2(context.Background(), fileUpload.BucketName, "example.txt")
	require.NoError(b, err)

	// download file chunks
	err = sdk.Download(context.Background(), fileDownload, &downloaded)
	require.NoError(b, err)

	assert.Equal(b, len(expected), len(downloaded.Bytes()))
	assert.EqualValues(b, expected[:10], downloaded.Bytes()[:10])
	assert.EqualValues(b, expected[len(expected)-10:], downloaded.Bytes()[len(downloaded.Bytes())-10:])
}
