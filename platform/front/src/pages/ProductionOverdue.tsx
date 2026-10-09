import React from 'react';
import Overdue from './Overdue';

// 生产逾期订单 - 继承 Overdue，隐藏费用字段
const ProductionOverdue: React.FC<any> = (props) => {
  return <Overdue {...props} hideCostFields={true} />;
};

export default ProductionOverdue;