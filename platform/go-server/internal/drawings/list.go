package drawings

// CountSQL adds only indexed metadata lookups, never file contents or an extra list request.
// A migrated legacy image must not reappear after all its versions are archived.
func CountSQL(alias string) string {
	return "(SELECT COALESCE(SUM(pd.is_current),0) + IF(COALESCE(" + alias + ".drawing_data,'')<>'' AND COALESCE(MAX(pd.legacy_source),0)=0,1,0) FROM part_drawings pd WHERE pd.corp_id=" + alias + ".corp_id AND pd.order_item_id=" + alias + ".id)"
}
