-- 1. 创建数据库（如果还没创建）
-- CREATE DATABASE gamenft;

-- 2. 创建专用用户（推荐）
CREATE USER admin1 WITH PASSWORD 'showmethemoney';
GRANT ALL PRIVILEGES ON DATABASE gamenft TO admin1;
