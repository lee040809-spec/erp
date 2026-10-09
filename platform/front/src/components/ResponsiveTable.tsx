import React, { Children, cloneElement, isValidElement, ReactNode } from 'react';

type SectionProps = { children?: ReactNode };
type CellProps = SectionProps & { colSpan?: number; 'data-label'?: string; 'data-full-width'?: boolean };

function headerText(node: ReactNode): string {
    return Children.toArray(node).map(child => {
        if (typeof child === 'string' || typeof child === 'number') return String(child);
        return isValidElement<SectionProps>(child) ? headerText(child.props.children) : '';
    }).join('');
}

// Use the same rows and values on both screens; phones display labelled cards.
export default function ResponsiveTable({ children, className = '', ...props }: React.TableHTMLAttributes<HTMLTableElement>) {
    const sections = Children.toArray(children);
    const head = sections.find(child => isValidElement(child) && child.type === 'thead');
    const headerRow = isValidElement<SectionProps>(head)
        ? Children.toArray(head.props.children).find(child => isValidElement(child) && child.type === 'tr')
        : undefined;
    const labels = isValidElement<SectionProps>(headerRow)
        ? Children.toArray(headerRow.props.children).map(child => isValidElement<SectionProps>(child) ? headerText(child.props.children) : '')
        : [];

    function labelRows(rows: ReactNode): ReactNode {
        return Children.map(rows, row => {
            if (!isValidElement<SectionProps>(row)) return row;
            if (row.type === React.Fragment) return cloneElement(row, { children: labelRows(row.props.children) });
            if (row.type !== 'tr') return row;
            return cloneElement(row, {
                children: Children.map(row.props.children, (cell, index) => {
                    if (!isValidElement<CellProps>(cell) || cell.type !== 'td') return cell;
                    return cloneElement(cell, {
                        'data-label': cell.props.colSpan && cell.props.colSpan > 1 ? undefined : labels[index],
                        'data-full-width': Boolean(cell.props.colSpan && cell.props.colSpan > 1)
                    });
                })
            });
        });
    }

    return <table {...props} className={`erp-responsive-table ${className}`}>
        {Children.map(children, section => {
            if (!isValidElement<SectionProps>(section) || section.type !== 'tbody') return section;
            return cloneElement(section, {
                children: labelRows(section.props.children)
            });
        })}
    </table>;
}
