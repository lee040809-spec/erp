import React, {
  Children,
  cloneElement,
  isValidElement,
  ReactNode,
} from "react";

type SectionProps = { children?: ReactNode };
type CellProps = SectionProps & {
  colSpan?: number;
  "data-label"?: string;
  "data-full-width"?: boolean;
};

function headerText(node: ReactNode): string {
  return Children.toArray(node)
    .map((child) => {
      if (typeof child === "string" || typeof child === "number")
        return String(child);
      return isValidElement<SectionProps>(child)
        ? headerText(child.props.children)
        : "";
    })
    .join("");
}

function flatCells(children: ReactNode): ReactNode[] {
  return Children.toArray(children).flatMap((child) =>
    isValidElement<SectionProps>(child) && child.type === React.Fragment
      ? flatCells(child.props.children)
      : [child],
  );
}

// Use the same rows and values on both screens; phones display labelled cards.
export default function ResponsiveTable({
  children,
  className = "",
  ...props
}: React.TableHTMLAttributes<HTMLTableElement>) {
  const sections = Children.toArray(children);
  const head = sections.find(
    (child) => isValidElement(child) && child.type === "thead",
  );
  const headerRow = isValidElement<SectionProps>(head)
    ? Children.toArray(head.props.children).find(
        (child) => isValidElement(child) && child.type === "tr",
      )
    : undefined;
  const labels = isValidElement<SectionProps>(headerRow)
    ? flatCells(headerRow.props.children).map((child) =>
        isValidElement<SectionProps>(child)
          ? headerText(child.props.children)
          : "",
      )
    : [];

  function labelRows(rows: ReactNode): ReactNode {
    return Children.map(rows, (row) => {
      if (!isValidElement<SectionProps>(row)) return row;
      if (row.type === React.Fragment)
        return cloneElement(row, { children: labelRows(row.props.children) });
      if (row.type !== "tr") return row;
      let column = 0;
      return cloneElement(row, {
        children: Children.map(flatCells(row.props.children), (cell) => {
          if (!isValidElement<CellProps>(cell) || cell.type !== "td")
            return cell;
          const label = labels[column];
          column += cell.props.colSpan || 1;
          return cloneElement(cell, {
            "data-label":
              cell.props.colSpan && cell.props.colSpan > 1 ? undefined : label,
            "data-full-width": Boolean(
              cell.props.colSpan && cell.props.colSpan > 1,
            ),
          });
        }),
      });
    });
  }

  return (
    <table {...props} className={`erp-responsive-table ${className}`}>
      {Children.map(children, (section) => {
        if (!isValidElement<SectionProps>(section) || section.type !== "tbody")
          return section;
        return cloneElement(section, {
          children: labelRows(section.props.children),
        });
      })}
    </table>
  );
}
