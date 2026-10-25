# 资源表 page_resources 字段格式说明

## 概述

`page_resources` 字段用于存储菜单资源下的内部页面资源（如按钮、链接等），使用 MySQL JSON 类型存储。

## 字段结构

```json
[
  {
    "key": "btn-preview-delivery-note",
    "name": "预览送货单",
    "type": "button"
  },
  {
    "key": "btn-print-order",
    "name": "打印订单",
    "type": "button"
  }
]
```

## 属性说明

| 属性 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `key` | string | 是 | 资源唯一标识，全局唯一，用于权限控制和模板绑定 |
| `name` | string | 是 | 资源显示名称 |
| `type` | string | 是 | 资源类型：`button`（按钮）、`link`（链接）、`other`（其他） |

## 使用场景

### 1. 按钮权限控制

通过 `key` 标识页面内的按钮，实现细粒度的权限控制：

```typescript
// 检查用户是否有某个按钮的权限
const hasButtonPermission = (buttonKey: string) => {
  const permissions = user.permissions;
  return permissions.includes(buttonKey);
};
```

### 2. 打印模板绑定

打印模板通过 `button_key` 字段与按钮资源关联：

```sql
-- 创建打印模板时，button_key 对应 page_resources 中的 key
INSERT INTO print_templates (name, button_key, button_name)
VALUES ('送货单模板', 'btn-preview-delivery-note', '预览送货单');
```

## 命名规范

### key 命名规范

- 格式：`{类型前缀}-{动作}-{对象}`
- 使用小写字母和连字符
- 全局唯一

示例：
- `btn-preview-delivery-note` - 预览送货单按钮
- `btn-print-order` - 打印订单按钮
- `btn-export-excel` - 导出Excel按钮
- `link-view-detail` - 查看详情链接

### type 类型说明

| 类型 | 说明 | 示例 |
|------|------|------|
| `button` | 页面内的操作按钮 | 提交、删除、打印、导出等 |
| `link` | 页面内的跳转链接 | 查看详情、编辑等 |
| `other` | 其他页面资源 | 自定义组件权限等 |

## 数据库操作示例

### 插入带 page_resources 的菜单资源

```sql
INSERT INTO resources (
  resource_type,
  resource_key,
  name,
  path,
  icon,
  sort_order,
  page_resources,
  platform_type,
  status
) VALUES (
  'menu',
  'delivery',
  '送货管理',
  'delivery',
  'Truck',
  15,
  '[{"key": "btn-preview-delivery-note", "name": "预览送货单", "type": "button"}]',
  'business',
  'active'
);
```

### 查询 page_resources

```sql
SELECT id, resource_key, name, page_resources
FROM resources
WHERE JSON_LENGTH(page_resources) > 0;
```

### 更新 page_resources

```sql
UPDATE resources
SET page_resources = JSON_ARRAY_APPEND(
  page_resources,
  '$',
  JSON_OBJECT('key', 'btn-export-excel', 'name', '导出Excel', 'type', 'button')
)
WHERE resource_key = 'orders';
```

## 前端使用示例

### TypeScript 类型定义

```typescript
interface PageResource {
  key: string;
  name: string;
  type: 'button' | 'link' | 'other';
}

interface Resource {
  id?: number;
  resource_key: string;
  name: string;
  page_resources?: PageResource[];
  // ... 其他字段
}
```

### 权限检查组件

```tsx
const ButtonWithPermission: React.FC<{
  buttonKey: string;
  children: React.ReactNode;
  onClick: () => void;
}> = ({ buttonKey, children, onClick }) => {
  const { hasPermission } = usePermissions();

  if (!hasPermission(buttonKey)) {
    return null;
  }

  return (
    <button onClick={onClick}>
      {children}
    </button>
  );
};

// 使用
<ButtonWithPermission buttonKey="btn-preview-delivery-note" onClick={handlePreview}>
  预览送货单
</ButtonWithPermission>
```