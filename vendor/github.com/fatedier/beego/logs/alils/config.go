package alils

const (
	version         = "0.5.0"     // SDK version
	signatureMethod = "hmac-sha1" // Signature method

	// OffsetNewest stands for the log head offset, i.e. the offset that will be
	// assigned to the next message that will be produced to the shard.
	OffsetNewest = "end"
	// OffsetOldest stands for the oldest offset available on the logstore for a
	// shard.
	OffsetOldest = "begin"
)
