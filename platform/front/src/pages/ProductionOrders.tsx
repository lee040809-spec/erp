import React from 'react';
import Orders from './Orders';

// 生产订单管理 - 继承 Orders，隐藏费用字段
const ProductionOrders: React.FC<any> = (props) => {
  return <Orders {...props} hideCostFields={true} />;
};

export default ProductionOrders;