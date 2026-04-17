// Package meta 提供证书签发所需的元数据接口：
// 1. 主体预置字段（基于 XCA dn.txt）
// 2. 预置 OID 库（基于 XCA oids.txt，按分类组织）
//
// 这些元数据供前端 UI 下拉选择/表单验证使用，不进入数据库。
// 后端仅静态返回常量数据，性能零开销。
package meta

// SubjectField 描述一个主体 DN 字段。
type SubjectField struct {
	Name      string `json:"name"`       // 字段标识（如 "CN"/"emailAddress"）
	Display   string `json:"display"`    // 中文显示名
	Required  bool   `json:"required"`   // 是否必填
	MaxLength int    `json:"max_length"` // 最大长度（0 = 不限制）
	Pattern   string `json:"pattern"`    // 可选正则校验
}

// subjectFields 是预置的主体字段列表（覆盖 RFC 5280 + XCA dn.txt）。
var subjectFields = []SubjectField{
	{Name: "C", Display: "国家", MaxLength: 2, Pattern: "^[A-Z]{2}$"},
	{Name: "ST", Display: "省/州", MaxLength: 128},
	{Name: "L", Display: "城市", MaxLength: 128},
	{Name: "O", Display: "组织", MaxLength: 64},
	{Name: "OU", Display: "部门", MaxLength: 64},
	{Name: "CN", Display: "通用名", Required: true, MaxLength: 64},
	{Name: "emailAddress", Display: "邮箱地址", MaxLength: 128},
	{Name: "serialNumber", Display: "序列号", MaxLength: 64},
	{Name: "givenName", Display: "名", MaxLength: 64},
	{Name: "surname", Display: "姓", MaxLength: 64},
	{Name: "title", Display: "头衔", MaxLength: 64},
	{Name: "initials", Display: "姓名首字母", MaxLength: 16},
	{Name: "description", Display: "描述", MaxLength: 256},
	{Name: "role", Display: "角色", MaxLength: 64},
	{Name: "pseudonym", Display: "假名", MaxLength: 128},
	{Name: "name", Display: "名称", MaxLength: 128},
	{Name: "dnQualifier", Display: "DN 限定符", MaxLength: 64},
	{Name: "generationQualifier", Display: "代际限定符", MaxLength: 32},
	{Name: "x500UniqueIdentifier", Display: "X.500 唯一标识符", MaxLength: 64},
	{Name: "businessCategory", Display: "商业分类", MaxLength: 128},
	{Name: "streetAddress", Display: "街道地址", MaxLength: 128},
	{Name: "localityName", Display: "地区名称", MaxLength: 128},
	{Name: "postalCode", Display: "邮政编码", MaxLength: 32},
	{Name: "IncLocalityName", Display: "注册地城市", MaxLength: 128},
	{Name: "IncStateOrProvinceName", Display: "注册地省份", MaxLength: 128},
	{Name: "IncCountryName", Display: "注册地国家", MaxLength: 2, Pattern: "^[A-Z]{2}$"},
}

// GetSubjectFields 返回预置的主体字段列表（按 dn.txt 顺序）。
func GetSubjectFields() []SubjectField {
	// 返回副本避免外部修改内置数据
	out := make([]SubjectField, len(subjectFields))
	copy(out, subjectFields)
	return out
}
