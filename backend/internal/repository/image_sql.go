package repository

// ImageJSON returns a SQL expression yielding the model.Image JSON for the public file id in column, or NULL.
// column must be a constant identifier such as "p.og_image_file_id", never user input.
func ImageJSON(column string) string {
	return `(SELECT json_build_object('id', f.id, 'storage_key', f.storage_key, 'mime_type', f.mime_type,
		'width', coalesce(f.width, 0), 'height', coalesce(f.height, 0), 'variants', f.variants)
		FROM files f WHERE f.id = ` + column + ` AND f.deleted_at IS NULL AND f.is_public)`
}
