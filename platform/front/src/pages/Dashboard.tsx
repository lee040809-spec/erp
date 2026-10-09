import React from 'react';
import ProductionWorkbench from './ProductionWorkbench';
const Dashboard: React.FC<any> = props => <ProductionWorkbench {...props} orderView="orders" />;
export default Dashboard;
