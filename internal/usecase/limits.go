package usecase

const (
	MaxWishlistsPerUser    = 20
	MaxPresentsPerWishlist = 100
	MaxBlocksPerWishlist   = 100
	MaxBulkUploadFiles     = 10
	MaxFileSize            = 10 * 1024 * 1024 // 10MB

	MaxTitleLen       = 200
	MaxDescriptionLen = 2000
	MaxURLLen         = 2048
	MaxBlockDataSize  = 10 * 1024 // 10KB per block (raw JSON bytes)
	MaxBlockTextField = 5000      // chars for text/quote/checklist content
)
