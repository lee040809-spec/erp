import React from 'react';
import Warning from './Warning';

// 生产告警订单 - 继承 Warning，隐藏费用字段
const ProductionWarning: React.FC<any> = (props) => {
  return <Warning {...props} hideCostFields={true} />;
};

export default ProductionWarning;