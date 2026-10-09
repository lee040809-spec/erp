import "dotenv/config";
import express from "express";
import { createServer as createViteServer } from "vite";
import mysql from "mysql2/promise";
import path from "path";
import fs from "fs";
import crypto from "crypto";

function logStatus(msg: string) {
  try {
    fs.appendFileSync('status_debug.log', `[${new Date().toISOString()}] ${msg}\n`);
  } catch (err) {
    console.error("Failed to write to status_debug.log", err);
  }
}

// Login configuration
const ADMIN_USER = "admin";
const ADMIN_PASSWORD = "yhs@2026";  // 裕合森
const MAX_LOGIN_ATTEMPTS = 3;
const LOCK_DURATION_MS = 2 * 60 * 60 * 1000; // 2 hours

// Login state (in-memory)
interface LoginState {
  attempts: number;
  lockedUntil: Date | null;
}

const loginState: LoginState = {
  attempts: 0,
  lockedUntil: null
};

// Generate session token
function generateToken(): string {
  return crypto.randomBytes(32).toString('hex');
}

// Active sessions
const activeSessions = new Set<string>();

// Helper to convert undefined to null for MySQL
function toNull<T>(value: T | undefined | null): T | null {
  return value === undefined ? null : value ?? null;
}

// Helper to convert date value to MySQL compatible format (null for empty/invalid)
function toDate(value: any): string | null {
  if (!value || value === '' || value === '-') return null;
  if (typeof value === 'string' && value.match(/^\d{4}-\d{2}-\d{2}$/)) return value;
  return null;
}

// Helper to convert MySQL DECIMAL strings to numbers
function toNumber(value: any): number {
  if (value === null || value === undefined) return 0;
  return Number(value);
}

// Helper to format date to YYYY-MM-DD (handle timezone properly)
function formatDate(date: any): string | null {
  if (!date) return null;
  // Already in correct format
  if (typeof date === 'string' && date.match(/^\d{4}-\d{2}-\d{2}$/)) return date;
  // ISO string format like "2026-02-27T16:00:00.000Z" - need to add 1 day for timezone offset
  if (typeof date === 'string' && date.includes('T')) {
    const d = new Date(date);
    // Add timezone offset to get correct local date
    d.setMinutes(d.getMinutes() + d.getTimezoneOffset());
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }
  // Date object
  const d = new Date(date);
  if (isNaN(d.getTime())) return null;
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

// Helper to convert order item with numeric fields
function convertOrderItem(item: any): any {
  return {
    ...item,
    quantity: toNumber(item.quantity),
    scrap_quantity: toNumber(item.scrap_quantity),
    unit_price: toNumber(item.unit_price),
    total_price: toNumber(item.total_price),
    delivered_quantity: toNumber(item.delivered_quantity),
    tool_cost: toNumber(item.tool_cost),
    fixture_cost: toNumber(item.fixture_cost),
    material_cost: toNumber(item.material_cost),
    other_cost: toNumber(item.other_cost),
    start_date: formatDate(item.start_date),
    due_date: formatDate(item.due_date),
    completion_date: formatDate(item.completion_date),
    processes: (item.processes || []).map((p: any) => ({
      ...p,
      outsourcing_fee: toNumber(p.outsourcing_fee),
      is_outsourced: Number(p.is_outsourced) || 0
    }))
  };
}

// Helper to convert order with date formatting
function convertOrder(order: any): any {
  return {
    ...order,
    start_date: formatDate(order.start_date),
    due_date: formatDate(order.due_date)
  };
}

// MySQL connection pool
const pool = mysql.createPool({
  host: process.env.MYSQL_HOST || "localhost",
  user: process.env.MYSQL_USER || "root",
  password: process.env.MYSQL_PASSWORD || "",
  database: process.env.MYSQL_DATABASE || "erp",
  port: Number(process.env.MYSQL_PORT) || 3306,
  waitForConnections: true,
  connectionLimit: 10,
  queueLimit: 0
});

// Initialize Database Schema
async function initDatabase() {
  const connection = await pool.getConnection();
  try {
    await connection.execute(`
      CREATE TABLE IF NOT EXISTS customers (
        id INT AUTO_INCREMENT PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        contact VARCHAR(255),
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
      )
    `);

    await connection.execute(`
      CREATE TABLE IF NOT EXISTS orders (
        id INT AUTO_INCREMENT PRIMARY KEY,
        customer_id INT,
        customer_name VARCHAR(255),
        order_number VARCHAR(255),
        status VARCHAR(50) DEFAULT 'pending',
        priority VARCHAR(50) DEFAULT 'medium',
        start_date DATE,
        due_date DATE,
        notes TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
      )
    `);

    await connection.execute(`
      CREATE TABLE IF NOT EXISTS order_items (
        id INT AUTO_INCREMENT PRIMARY KEY,
        order_id INT,
        part_name VARCHAR(255) NOT NULL,
        part_number VARCHAR(255),
        quantity INT DEFAULT 1,
        scrap_quantity INT DEFAULT 0,
        unit_price DECIMAL(10,2),
        total_price DECIMAL(10,2),
        status VARCHAR(50) DEFAULT 'pending',
        drawing_data LONGTEXT,
        notes TEXT,
        completion_date DATE,
        start_date DATE,
        due_date DATE,
        delivered_quantity INT,
        tool_cost DECIMAL(10,2),
        fixture_cost DECIMAL(10,2),
        material_cost DECIMAL(10,2),
        other_cost DECIMAL(10,2),
        item_notes TEXT,
        FOREIGN KEY (order_id) REFERENCES orders(id)
      )
    `);

    await connection.execute(`
      CREATE TABLE IF NOT EXISTS order_processes (
        id INT AUTO_INCREMENT PRIMARY KEY,
        order_item_id INT,
        name VARCHAR(255) NOT NULL,
        is_outsourced TINYINT DEFAULT 0,
        outsourcing_fee DECIMAL(10,2) DEFAULT 0,
        status VARCHAR(50) DEFAULT 'pending',
        sort_order INT DEFAULT 0,
        FOREIGN KEY (order_item_id) REFERENCES order_items(id)
      )
    `);

    await connection.execute(`
      CREATE TABLE IF NOT EXISTS materials (
        id INT AUTO_INCREMENT PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        spec VARCHAR(255),
        quantity DECIMAL(10,2) DEFAULT 0,
        unit VARCHAR(50) DEFAULT 'kg'
      )
    `);

    await connection.execute(`
      CREATE TABLE IF NOT EXISTS remnants (
        id INT AUTO_INCREMENT PRIMARY KEY,
        material_id INT,
        dimensions VARCHAR(255),
        photo_data LONGTEXT,
        notes TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (material_id) REFERENCES materials(id)
      )
    `);

    await connection.execute(`
      CREATE TABLE IF NOT EXISTS advent_rules (
        id INT AUTO_INCREMENT PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        description TEXT,
        formula TEXT NOT NULL,
        target_status VARCHAR(50) DEFAULT 'pending',
        scopeType VARCHAR(50) DEFAULT 'general',
        ruleType VARCHAR(50) DEFAULT 'imminent',
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
      )
    `);

    // Seed Data
    await connection.execute(`
      INSERT IGNORE INTO customers (id, name, contact) VALUES (1, '大疆创新', '张工 13800138000')
    `);
    await connection.execute(`
      INSERT IGNORE INTO customers (id, name, contact) VALUES (2, '华为技术', '李工 13900139000')
    `);

    await connection.execute(`
      INSERT IGNORE INTO materials (id, name, spec, quantity, unit) VALUES (1, '45# 钢', 'Φ30', 50, 'kg')
    `);
    await connection.execute(`
      INSERT IGNORE INTO materials (id, name, spec, quantity, unit) VALUES (2, '6061 铝', '100*100', 20, '块')
    `);
    await connection.execute(`
      INSERT IGNORE INTO materials (id, name, spec, quantity, unit) VALUES (3, 'H59 黄铜', 'Φ20', 15, 'kg')
    `);

    // Seed Orders and Items
    await connection.execute(`
      INSERT IGNORE INTO orders (id, customer_id, order_number, status, priority, due_date)
      VALUES (1, 1, 'ORD-20240311-001', 'processing', 'high', '2024-03-20')
    `);

    await connection.execute(`
      INSERT IGNORE INTO order_items (id, order_id, part_name, part_number, quantity, unit_price, total_price, status)
      VALUES (1, 1, '主轴连接件', 'DJ-001-A', 5, 120.00, 600.00, 'processing')
    `);

    await connection.execute(`
      INSERT IGNORE INTO order_processes (id, order_item_id, name, status, sort_order)
      VALUES (1, 1, '下料', 'completed', 1)
    `);
    await connection.execute(`
      INSERT IGNORE INTO order_processes (id, order_item_id, name, status, sort_order)
      VALUES (2, 1, '车', 'processing', 2)
    `);
    await connection.execute(`
      INSERT IGNORE INTO order_processes (id, order_item_id, name, status, sort_order)
      VALUES (3, 1, '铣', 'pending', 3)
    `);

    await connection.execute(`
      INSERT IGNORE INTO orders (id, customer_id, order_number, status, priority, due_date)
      VALUES (2, 2, 'ORD-20240311-002', 'pending', 'medium', '2024-03-25')
    `);

    await connection.execute(`
      INSERT IGNORE INTO order_items (id, order_id, part_name, part_number, quantity, unit_price, total_price, status)
      VALUES (2, 2, '散热片', 'HW-99-S', 100, 15.50, 1550.00, 'pending')
    `);

    await connection.execute(`
      INSERT IGNORE INTO order_processes (id, order_item_id, name, status, sort_order)
      VALUES (4, 2, '铣', 'pending', 1)
    `);
    await connection.execute(`
      INSERT IGNORE INTO order_processes (id, order_item_id, name, status, sort_order)
      VALUES (5, 2, '表面处理', 'pending', 2)
    `);

    console.log("Database initialized successfully");
  } finally {
    connection.release();
  }
}

// Run migrations
async function runMigrations() {
  const migrations = [
    "ALTER TABLE orders ADD COLUMN start_date DATE",
    "ALTER TABLE order_items ADD COLUMN start_date DATE",
    "ALTER TABLE order_items ADD COLUMN due_date DATE",
    "ALTER TABLE order_items ADD COLUMN completion_date DATE",
    "ALTER TABLE order_items ADD COLUMN delivered_quantity INT",
    "ALTER TABLE order_items ADD COLUMN tool_cost DECIMAL(10,2)",
    "ALTER TABLE order_items ADD COLUMN fixture_cost DECIMAL(10,2)",
    "ALTER TABLE order_items ADD COLUMN material_cost DECIMAL(10,2)",
    "ALTER TABLE order_items ADD COLUMN other_cost DECIMAL(10,2)",
    "ALTER TABLE order_items ADD COLUMN scrap_quantity INT DEFAULT 0",
    "ALTER TABLE order_items ADD COLUMN item_notes TEXT",
    "ALTER TABLE advent_rules ADD COLUMN target_status VARCHAR(50) DEFAULT 'pending'",
    "ALTER TABLE advent_rules ADD COLUMN scopeType VARCHAR(50) DEFAULT 'general'",
    "ALTER TABLE advent_rules ADD COLUMN ruleType VARCHAR(50) DEFAULT 'imminent'",
    "ALTER TABLE orders ADD COLUMN customer_name VARCHAR(255)",
  ];

  for (const migration of migrations) {
    try {
      await pool.execute(migration);
    } catch (e) {
      // Ignore errors (e.g., column already exists)
    }
  }

  // Ensure scrap_quantity column has default values for existing rows
  try {
    await pool.execute("UPDATE order_items SET scrap_quantity = 0 WHERE scrap_quantity IS NULL");
  } catch (e) {
    // Ignore if column doesn't exist
  }
}

function calculateStatus(subItems: any[], label: string = 'unknown'): string {
  if (!subItems || subItems.length === 0) {
    logStatus(`calculateStatus(${label}): No items, returning pending`);
    return 'pending';
  }

  const statuses = subItems.map(s => s.status || 'pending');
  let result = 'processing';

  if (statuses.every(s => s === 'completed' || s === 'delivered')) {
    result = 'completed';
  } else if (statuses.every(s => s === 'pending')) {
    result = 'pending';
  }

  logStatus(`calculateStatus(${label}): Input statuses: ${JSON.stringify(statuses)} -> Result: ${result}`);
  return result;
}

async function startServer() {
  // Initialize database
  await initDatabase();
  await runMigrations();

  const app = express();
  const PORT = Number(process.env.PORT) || 3000;

  app.use(express.json({ limit: '50mb' }));

  app.get("/api/health", (req, res) => {
    res.json({ status: "ok", version: "mysql", time: new Date().toISOString() });
  });

  // Auth API Routes
  app.post("/api/login", (req, res) => {
    const { username, password } = req.body;

    // Check if account is locked
    if (loginState.lockedUntil && new Date() < loginState.lockedUntil) {
      const remainingMs = loginState.lockedUntil.getTime() - Date.now();
      const remainingMinutes = Math.ceil(remainingMs / 60000);
      return res.status(403).json({
        success: false,
        error: "账户已锁定",
        locked: true,
        remainingMinutes
      });
    }

    // Check if lock has expired
    if (loginState.lockedUntil && new Date() >= loginState.lockedUntil) {
      loginState.attempts = 0;
      loginState.lockedUntil = null;
    }

    // Validate credentials
    if (username === ADMIN_USER && password === ADMIN_PASSWORD) {
      // Reset attempts on successful login
      loginState.attempts = 0;
      loginState.lockedUntil = null;

      // Generate session token
      const token = generateToken();
      activeSessions.add(token);

      res.json({
        success: true,
        token,
        user: { username: ADMIN_USER }
      });
    } else {
      // Increment failed attempts
      loginState.attempts++;
      const remainingAttempts = MAX_LOGIN_ATTEMPTS - loginState.attempts;

      if (loginState.attempts >= MAX_LOGIN_ATTEMPTS) {
        // Lock the account
        loginState.lockedUntil = new Date(Date.now() + LOCK_DURATION_MS);
        return res.status(403).json({
          success: false,
          error: "登录失败次数过多，账户已锁定2小时",
          locked: true,
          remainingMinutes: 120
        });
      }

      res.status(401).json({
        success: false,
        error: `账号或密码错误，还剩 ${remainingAttempts} 次登录机会`,
        remainingAttempts
      });
    }
  });

  // Check auth status
  app.get("/api/auth/status", (req, res) => {
    const token = req.headers.authorization?.replace('Bearer ', '');

    // Check if account is locked
    if (loginState.lockedUntil && new Date() < loginState.lockedUntil) {
      const remainingMs = loginState.lockedUntil.getTime() - Date.now();
      const remainingMinutes = Math.ceil(remainingMs / 60000);
      return res.json({
        authenticated: false,
        locked: true,
        remainingMinutes
      });
    }

    if (token && activeSessions.has(token)) {
      res.json({
        authenticated: true,
        user: { username: ADMIN_USER }
      });
    } else {
      res.json({ authenticated: false });
    }
  });

  // Logout
  app.post("/api/logout", (req, res) => {
    const token = req.headers.authorization?.replace('Bearer ', '');
    if (token) {
      activeSessions.delete(token);
    }
    res.json({ success: true });
  });

  // API Routes

  // Auth middleware for protected routes
  const authMiddleware = (req: express.Request, res: express.Response, next: express.NextFunction) => {
    const token = req.headers.authorization?.replace('Bearer ', '');

    // Check if account is locked
    if (loginState.lockedUntil && new Date() < loginState.lockedUntil) {
      return res.status(403).json({ error: "账户已锁定", locked: true });
    }

    if (!token || !activeSessions.has(token)) {
      return res.status(401).json({ error: "未授权访问" });
    }

    next();
  };

  // Apply auth middleware to all /api routes except auth routes
  app.use('/api', (req, res, next) => {
    // Skip auth for login and auth status routes
    if (req.path === '/login' || req.path === '/auth/status') {
      return next();
    }
    authMiddleware(req, res, next);
  });

  // Customers
  app.get("/api/customers", async (req, res) => {
    const [rows] = await pool.execute("SELECT * FROM customers ORDER BY name");
    res.json(rows);
  });

  app.post("/api/customers", async (req, res) => {
    const { name, contact } = req.body;
    const [result] = await pool.execute(
      "INSERT INTO customers (name, contact) VALUES (?, ?)",
      [toNull(name), toNull(contact)]
    ) as [mysql.ResultSetHeader, any];
    res.json({ id: result.insertId });
  });

  app.patch("/api/customers/:id", async (req, res) => {
    const { id } = req.params;
    const { name, contact } = req.body;
    await pool.execute("UPDATE customers SET name = ?, contact = ? WHERE id = ?", [toNull(name), toNull(contact), id]);
    res.json({ success: true });
  });

  app.delete("/api/customers/:id", async (req, res) => {
    const { id } = req.params;
    await pool.execute("DELETE FROM customers WHERE id = ?", [id]);
    res.json({ success: true });
  });

  // Orders
  app.get("/api/orders", async (req, res) => {
    const [orders] = await pool.execute(`
      SELECT orders.*,
        COALESCE(orders.customer_name, customers.name) as customer_name
      FROM orders
      LEFT JOIN customers ON orders.customer_id = customers.id
      ORDER BY start_date ASC
    `);

    // Attach items and processes to each order
    const fullOrders = await Promise.all((orders as any[]).map(async (order) => {
      const [items] = await pool.execute("SELECT * FROM order_items WHERE order_id = ? ORDER BY due_date ASC", [order.id]);
      const itemsWithProcesses = await Promise.all((items as any[]).map(async (item) => {
        const [processes] = await pool.execute("SELECT * FROM order_processes WHERE order_item_id = ? ORDER BY sort_order ASC", [item.id]);
        return convertOrderItem({ ...item, processes });
      }));
      return convertOrder({ ...order, items: itemsWithProcesses });
    }));

    res.json(fullOrders);
  });

  app.post("/api/orders", async (req, res) => {
    const { customer_id, customer_name, order_number, priority, start_date, due_date, notes, items } = req.body;

    if (!customer_id || !start_date || !due_date) {
      return res.status(400).send("Missing customer_id, start_date or due_date");
    }

    let finalOrderNumber = order_number;
    if (!finalOrderNumber) {
      const today = new Date().toISOString().slice(0, 10).replace(/-/g, '');
      const prefix = `YHS-${today}-`;
      const [ordersToday] = await pool.execute("SELECT order_number FROM orders WHERE order_number LIKE ?", [`${prefix}%`]);
      let maxSuffix = 0;
      (ordersToday as any[]).forEach(o => {
        const parts = o.order_number.split('-');
        const suffix = parseInt(parts[parts.length - 1]);
        if (!isNaN(suffix) && suffix > maxSuffix) maxSuffix = suffix;
      });
      finalOrderNumber = `${prefix}${maxSuffix + 1}`;
    }

    const connection = await pool.getConnection();
    try {
      await connection.beginTransaction();

      const [orderResult] = await connection.execute(
        "INSERT INTO orders (customer_id, customer_name, order_number, priority, start_date, due_date, notes) VALUES (?, ?, ?, ?, ?, ?, ?)",
        [toNull(customer_id), toNull(customer_name), finalOrderNumber, priority || 'medium', start_date, due_date, toNull(notes)]
      ) as [mysql.ResultSetHeader, any];
      const orderId = orderResult.insertId;

      // Insert items if provided
      if (items && Array.isArray(items)) {
        for (const item of items) {
          const total_price = (item.quantity || 1) * (item.unit_price || 0);
          const itemStatus = calculateStatus(item.processes || []);
          const [itemResult] = await connection.execute(
            `INSERT INTO order_items (
              order_id, part_name, part_number, quantity, scrap_quantity, unit_price, total_price,
              drawing_data, notes, status, completion_date, start_date, due_date, delivered_quantity,
              tool_cost, fixture_cost, material_cost, other_cost, item_notes
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
            [
              orderId, toNull(item.part_name), toNull(item.part_number), item.quantity || 1, toNull(item.scrap_quantity),
              item.unit_price || 0, total_price, toNull(item.drawing_data), toNull(item.notes), itemStatus,
              toDate(item.completion_date), toDate(item.start_date), toDate(item.due_date), toNull(item.delivered_quantity),
              toNull(item.tool_cost), toNull(item.fixture_cost), toNull(item.material_cost), toNull(item.other_cost), toNull(item.item_notes)
            ]
          ) as [mysql.ResultSetHeader, any];
          const itemId = itemResult.insertId;

          if (item.processes && Array.isArray(item.processes)) {
            for (let index = 0; index < item.processes.length; index++) {
              const p = item.processes[index];
              if (!p || !p.name) continue;
              await connection.execute(
                "INSERT INTO order_processes (order_item_id, name, is_outsourced, outsourcing_fee, status, sort_order) VALUES (?, ?, ?, ?, ?, ?)",
                [itemId, p.name, p.is_outsourced ? 1 : 0, toNull(p.outsourcing_fee) || 0, toNull(p.status) || 'pending', index]
              );
            }
          }
        }
      }

      // Calculate and update order status
      const [allItems] = await connection.execute("SELECT status FROM order_items WHERE order_id = ?", [orderId]);
      const orderStatus = calculateStatus(allItems as any[], `Initial Order ${orderId}`);
      logStatus(`Order ${orderId} created. Initial status: ${orderStatus}`);
      await connection.execute("UPDATE orders SET status = ? WHERE id = ?", [orderStatus, orderId]);

      await connection.commit();
      res.json({ id: orderId });
    } catch (err) {
      await connection.rollback();
      console.error("Error creating order:", err);
      res.status(500).json({ error: (err as Error).message });
    } finally {
      connection.release();
    }
  });

  app.patch("/api/orders/:id", async (req, res) => {
    const { id } = req.params;
    const { customer_id, customer_name, priority, start_date, due_date, notes, status, items } = req.body;

    if (!start_date || start_date === "" || !due_date || due_date === "") {
      return res.status(400).send("start_date and due_date cannot be empty");
    }

    const connection = await pool.getConnection();
    try {
      await connection.beginTransaction();

      // Update order fields
      if (status && !items) {
        // Simple status update
        await connection.execute("UPDATE orders SET status = ? WHERE id = ?", [status, id]);
      } else {
        // Full update
        await connection.execute(
          `UPDATE orders SET customer_id = ?, customer_name = ?, priority = ?, start_date = ?, due_date = ?, notes = ? WHERE id = ?`,
          [toNull(customer_id), toNull(customer_name), priority || 'medium', start_date, due_date, toNull(notes), id]
        );

        if (items && Array.isArray(items)) {
          // Delete existing items and processes
          const [existingItems] = await connection.execute("SELECT id FROM order_items WHERE order_id = ?", [id]);
          const itemIds = (existingItems as any[]).map(item => item.id);

          if (itemIds.length > 0) {
            const placeholders = itemIds.map(() => '?').join(',');
            await connection.execute(`DELETE FROM order_processes WHERE order_item_id IN (${placeholders})`, itemIds);
            await connection.execute("DELETE FROM order_items WHERE order_id = ?", [id]);
          }

          // Re-insert items
          for (const item of items) {
            const total_price = (item.quantity || 1) * (item.unit_price || 0);
            const itemStatus = calculateStatus(item.processes || []);
            const [itemResult] = await connection.execute(
              `INSERT INTO order_items (
                order_id, part_name, part_number, quantity, scrap_quantity, unit_price, total_price,
                drawing_data, notes, status, completion_date, start_date, due_date, delivered_quantity,
                tool_cost, fixture_cost, material_cost, other_cost, item_notes
              ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
              [
                id, toNull(item.part_name), toNull(item.part_number), item.quantity || 1, toNull(item.scrap_quantity),
                item.unit_price || 0, total_price, toNull(item.drawing_data), toNull(item.notes), itemStatus,
                toDate(item.completion_date), toDate(item.start_date), toDate(item.due_date), toNull(item.delivered_quantity),
                toNull(item.tool_cost), toNull(item.fixture_cost), toNull(item.material_cost), toNull(item.other_cost), toNull(item.item_notes)
              ]
            ) as [mysql.ResultSetHeader, any];
            const itemId = itemResult.insertId;

            if (item.processes && Array.isArray(item.processes)) {
              for (let index = 0; index < item.processes.length; index++) {
                const p = item.processes[index];
                if (!p || !p.name) continue;
                await connection.execute(
                  "INSERT INTO order_processes (order_item_id, name, is_outsourced, outsourcing_fee, status, sort_order) VALUES (?, ?, ?, ?, ?, ?)",
                  [itemId, p.name, p.is_outsourced ? 1 : 0, toNull(p.outsourcing_fee) || 0, p.status || 'pending', index]
                );
              }
            }
          }
        }
      }

      // Calculate and update order status
      const [allItems] = await connection.execute("SELECT status FROM order_items WHERE order_id = ?", [id]);
      const orderStatus = calculateStatus(allItems as any[], `Order ${id} Patch`);
      logStatus(`Order ${id} patched. New status: ${orderStatus}`);
      await connection.execute("UPDATE orders SET status = ? WHERE id = ?", [orderStatus, id]);

      await connection.commit();
      res.json({ success: true });
    } catch (err) {
      await connection.rollback();
      console.error("Error updating order:", err);
      res.status(500).json({ error: (err as Error).message });
    } finally {
      connection.release();
    }
  });

  app.patch("/api/order-items/:itemId", async (req, res) => {
    const { itemId } = req.params;
    const { status } = req.body;

    if (status) {
      await pool.execute("UPDATE order_items SET status = ? WHERE id = ?", [status, Number(itemId)]);

      // Auto-update order status
      const [itemRows] = await pool.execute("SELECT order_id FROM order_items WHERE id = ?", [Number(itemId)]) as [any[], any];
      if (itemRows.length > 0 && itemRows[0].order_id) {
        const orderId = itemRows[0].order_id;
        const [itemsInOrder] = await pool.execute("SELECT status FROM order_items WHERE order_id = ?", [Number(orderId)]);
        const newOrderStatus = calculateStatus(itemsInOrder as any[], `Order ${orderId} (Item ${itemId} update)`);
        logStatus(`Item ${itemId} status manual update. Order ${orderId} new status: ${newOrderStatus}`);
        await pool.execute("UPDATE orders SET status = ? WHERE id = ?", [newOrderStatus, Number(orderId)]);
      }
    }

    res.json({ success: true });
  });

  app.patch("/api/order-items/:itemId/processes/:processId", async (req, res) => {
    const { itemId, processId } = req.params;
    const { status, is_outsourced, outsourcing_fee } = req.body;

    let query = "UPDATE order_processes SET ";
    const params: any[] = [];
    const updates: string[] = [];

    if (status) {
      updates.push("status = ?");
      params.push(status);
    }
    if (is_outsourced !== undefined) {
      updates.push("is_outsourced = ?");
      params.push(is_outsourced ? 1 : 0);
    }
    if (outsourcing_fee !== undefined) {
      updates.push("outsourcing_fee = ?");
      params.push(toNull(outsourcing_fee));
    }

    if (updates.length === 0) return res.json({ success: true });

    query += updates.join(", ") + " WHERE id = ?";
    params.push(processId);

    const connection = await pool.getConnection();
    try {
      await connection.beginTransaction();

      logStatus(`Process ${processId} updated to ${status}. Re-calculating Item ${itemId}`);
      await connection.execute(query, params);
      logStatus(`Update Process OK`);

      const [allProcesses] = await connection.execute("SELECT id, status FROM order_processes WHERE order_item_id = ?", [Number(itemId)]);
      const newItemStatus = calculateStatus(allProcesses as any[], `Item ${itemId}`);
      await connection.execute("UPDATE order_items SET status = ? WHERE id = ?", [newItemStatus, Number(itemId)]);
      logStatus(`Update Item ${itemId} to ${newItemStatus}`);

      const [itemRows] = await connection.execute("SELECT order_id FROM order_items WHERE id = ?", [Number(itemId)]) as [any[], any];
      if (itemRows.length > 0 && itemRows[0].order_id) {
        const orderId = itemRows[0].order_id;
        const [itemsInOrder] = await connection.execute("SELECT id, status FROM order_items WHERE order_id = ?", [Number(orderId)]);
        const newOrderStatus = calculateStatus(itemsInOrder as any[], `Order ${orderId}`);
        logStatus(`Updating Order ${orderId} to ${newOrderStatus}`);
        await connection.execute("UPDATE orders SET status = ? WHERE id = ?", [newOrderStatus, Number(orderId)]);
      } else {
        logStatus(`WARNING: Could not find order_id for Item ${itemId}`);
      }

      await connection.commit();
      logStatus("Transaction committed successfully");
      res.json({ success: true });
    } catch (err) {
      await connection.rollback();
      console.error("Error updating process:", err);
      res.status(500).json({ error: (err as Error).message });
    } finally {
      connection.release();
    }
  });

  // Materials
  app.get("/api/materials", async (req, res) => {
    const [rows] = await pool.execute("SELECT * FROM materials");
    res.json(rows);
  });

  app.post("/api/materials", async (req, res) => {
    const { name, spec, quantity, unit } = req.body;
    const [result] = await pool.execute(
      "INSERT INTO materials (name, spec, quantity, unit) VALUES (?, ?, ?, ?)",
      [toNull(name), toNull(spec), toNull(quantity), toNull(unit)]
    ) as [mysql.ResultSetHeader, any];
    res.json({ id: result.insertId });
  });

  // Remnants
  app.get("/api/remnants", async (req, res) => {
    const [rows] = await pool.execute(`
      SELECT remnants.*, materials.name as material_name
      FROM remnants
      JOIN materials ON remnants.material_id = materials.id
    `);
    res.json(rows);
  });

  app.post("/api/remnants", async (req, res) => {
    const { material_id, dimensions, photo_data, notes } = req.body;
    const [result] = await pool.execute(
      "INSERT INTO remnants (material_id, dimensions, photo_data, notes) VALUES (?, ?, ?, ?)",
      [toNull(material_id), toNull(dimensions), toNull(photo_data), toNull(notes)]
    ) as [mysql.ResultSetHeader, any];
    res.json({ id: result.insertId });
  });

  // Finance / Reconciliation
  app.get("/api/finance/reconciliation", async (req, res) => {
    const [rows] = await pool.execute(`
      SELECT
        DATE_FORMAT(orders.due_date, '%Y-%m') as month,
        SUM(order_items.total_price) as total_amount,
        COUNT(DISTINCT orders.id) as order_count,
        SUM(CASE WHEN order_items.status = 'delivered' THEN order_items.total_price ELSE 0 END) as delivered_amount
      FROM orders
      JOIN order_items ON orders.id = order_items.order_id
      GROUP BY month
      ORDER BY month DESC
    `);
    res.json(rows);
  });

  // Advent Rules
  app.get("/api/advent-rules", async (req, res) => {
    const { name } = req.query;
    let query = "SELECT * FROM advent_rules";
    const params: any[] = [];

    if (name) {
      query += " WHERE name LIKE ?";
      params.push(`%${name}%`);
    }

    query += " ORDER BY created_at DESC";
    const [rows] = await pool.execute(query, params);
    res.json(rows);
  });

  app.post("/api/advent-rules", async (req, res) => {
    const { name, description, formula, target_status, scopeType, ruleType } = req.body;
    const [result] = await pool.execute(
      "INSERT INTO advent_rules (name, description, formula, target_status, scopeType, ruleType) VALUES (?, ?, ?, ?, ?, ?)",
      [toNull(name), toNull(description), toNull(formula), target_status || 'pending', scopeType || 'general', ruleType || 'imminent']
    ) as [mysql.ResultSetHeader, any];
    res.json({ id: result.insertId });
  });

  app.patch("/api/advent-rules/:id", async (req, res) => {
    const { id } = req.params;
    const { name, description, formula, target_status, scopeType, ruleType } = req.body;
    await pool.execute(
      "UPDATE advent_rules SET name = ?, description = ?, formula = ?, target_status = ?, scopeType = ?, ruleType = ? WHERE id = ?",
      [toNull(name), toNull(description), toNull(formula), toNull(target_status), toNull(scopeType), toNull(ruleType), id]
    );
    res.json({ success: true });
  });

  app.delete("/api/advent-rules/:id", async (req, res) => {
    const { id } = req.params;
    await pool.execute("DELETE FROM advent_rules WHERE id = ?", [id]);
    res.json({ success: true });
  });

  // Global error handler
  app.use((err: Error, _req: express.Request, res: express.Response, _next: express.NextFunction) => {
    console.error("Unhandled error:", err);
    res.status(500).json({ error: err.message || "Internal server error" });
  });

  // Vite middleware for development
  if (process.env.NODE_ENV !== "production") {
    const vite = await createViteServer({
      server: { middlewareMode: true },
      appType: "spa",
    });
    app.use(vite.middlewares);
  } else {
    app.use(express.static("dist"));
    app.get("*", (req, res) => {
      res.sendFile(path.resolve("dist/index.html"));
    });
  }

  app.listen(PORT, "0.0.0.0", () => {
    console.log(`Server running on http://localhost:${PORT}`);
  });
}

startServer().catch(console.error);