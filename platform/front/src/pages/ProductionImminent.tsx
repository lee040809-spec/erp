import React from 'react';
import Imminent from './Imminent';

// 生产临期订单 - 继承 Imminent，隐藏费用字段
const ProductionImminent: React.FC<any> = (props) => {
  return <Imminent {...props} hideCostFields={true} />;
};

export default ProductionImminent;