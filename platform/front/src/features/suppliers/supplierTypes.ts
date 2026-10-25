export const SUPPLIER_TYPES = [
  { value: "outsourcing", label: "外协" },
  { value: "material", label: "原材料" },
  { value: "tooling", label: "刀具" },
  { value: "fixture", label: "工装" },
  { value: "other", label: "其他" },
] as const;

export function supplierTypes(value?: string): string[] {
  return (value ?? "outsourcing").split(",").filter(Boolean);
}

export function supplierTypeLabels(value?: string): string {
  const selected = supplierTypes(value);
  return SUPPLIER_TYPES.filter((type) => selected.includes(type.value))
    .map((type) => type.label)
    .join("、");
}
