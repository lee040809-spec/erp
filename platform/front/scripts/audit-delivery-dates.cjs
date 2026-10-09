// Read-only audit: compare current stored dates and same-name parts across orders.
const mysql = require('mysql2/promise');
(async () => {
  const db = await mysql.createConnection({host:process.env.MYSQL_HOST,port:3306,user:process.env.MYSQL_USER,password:process.env.MYSQL_PASSWORD,database:process.env.MYSQL_DATABASE,dateStrings:true,connectTimeout:10000});
  try {
    const [target] = await db.execute(`SELECT o.corp_id,o.id order_id,o.order_number,DATE_FORMAT(o.due_date,'%Y-%m-%d') order_due_date,i.id item_id,i.part_name,DATE_FORMAT(i.due_date,'%Y-%m-%d') item_due_date FROM orders o JOIN order_items i ON i.order_id=o.id AND i.corp_id=o.corp_id WHERE o.order_number=? ORDER BY i.id`,['CGDD2609290043']);
    const [sameNames] = await db.execute(`SELECT o.corp_id,o.order_number,i.id item_id,i.part_name,DATE_FORMAT(i.due_date,'%Y-%m-%d') item_due_date FROM orders o JOIN order_items i ON i.order_id=o.id AND i.corp_id=o.corp_id WHERE i.part_name IN (SELECT x.part_name FROM order_items x JOIN orders q ON q.id=x.order_id AND q.corp_id=x.corp_id WHERE q.order_number=?) AND o.corp_id=? AND i.status NOT IN ('completed','delivered') ORDER BY o.order_number,i.id`,['CGDD2609290043',target[0]?.corp_id ?? -1]);
    const [active] = await db.execute(`SELECT o.order_number,i.id item_id,i.part_name,DATE_FORMAT(i.due_date,'%Y-%m-%d') item_due_date FROM orders o JOIN order_items i ON i.order_id=o.id AND i.corp_id=o.corp_id WHERE i.corp_id=? AND i.status NOT IN ('completed','delivered') ORDER BY i.id`,[target[0]?.corp_id ?? -1]);
    console.log(JSON.stringify({target,sameNames,active},null,2));
  } finally { await db.end(); }
})().catch(e=>{console.error('Read-only date audit failed:',e.code || e.name);process.exitCode=1;});
