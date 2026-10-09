import React from 'react';
import Delivery from './Delivery';

// 生产送货管理 - 继承 Delivery，显示费用字段
const ProductionDelivery: React.FC<any> = (props) => {
  return <Delivery {...props} hideCostFields={false} />;
};

export default ProductionDelivery;