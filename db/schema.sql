DROP TABLE IF EXISTS ingredients;
DROP TABLE IF EXISTS alloys;

CREATE TABLE alloys (
  id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  type ENUM('base','alloy','processed','raw_steel','final_steel') NOT NULL,
  raw_form_id VARCHAR(64) NULL,
  extra_ingredient_id VARCHAR(64) NULL,
  FOREIGN KEY (raw_form_id) REFERENCES alloys(id) ON DELETE SET NULL,
  FOREIGN KEY (extra_ingredient_id) REFERENCES alloys(id) ON DELETE SET NULL
);

CREATE TABLE ingredients (
  alloy_id VARCHAR(64) NOT NULL,
  ingredient_id VARCHAR(64) NOT NULL,
  min_pct FLOAT NOT NULL,
  max_pct FLOAT NOT NULL,
  PRIMARY KEY (alloy_id, ingredient_id),
  FOREIGN KEY (alloy_id) REFERENCES alloys(id) ON DELETE CASCADE,
  FOREIGN KEY (ingredient_id) REFERENCES alloys(id) ON DELETE CASCADE
);

-- 1) Insert ALL rows into `alloys` (including final_steel) before any `ingredients`.

-- Base metals
INSERT INTO alloys (id, name, type) VALUES
  ('copper', 'Copper', 'base'),
  ('zinc', 'Zinc', 'base'),
  ('bismuth', 'Bismuth', 'base'),
  ('silver', 'Silver', 'base'),
  ('gold', 'Gold', 'base'),
  ('nickel', 'Nickel', 'base'),
  ('pig_iron', 'Pig Iron', 'base');

-- Simple alloys (bronzes, brasses, etc.)
INSERT INTO alloys (id, name, type) VALUES
  ('bismuth_bronze', 'Bismuth Bronze', 'alloy'),
  ('black_bronze', 'Black Bronze', 'alloy'),
  ('brass', 'Brass', 'alloy'),
  ('rose_gold', 'Rose Gold', 'alloy'),
  ('sterling_silver', 'Sterling Silver', 'alloy');

-- Processed steel
INSERT INTO alloys (id, name, type) VALUES
  ('steel', 'Steel', 'processed');

-- Raw steels
INSERT INTO alloys (id, name, type) VALUES
  ('raw_black_steel', 'Raw Black Steel', 'raw_steel'),
  ('raw_blue_steel', 'Raw Blue Steel', 'raw_steel'),
  ('raw_red_steel', 'Raw Red Steel', 'raw_steel');

-- Final steels (depend on raw_steel rows inserted above)
INSERT INTO alloys (id, name, type, raw_form_id, extra_ingredient_id) VALUES
  ('black_steel', 'Black Steel', 'final_steel', 'raw_black_steel', 'pig_iron'),
  ('blue_steel', 'Blue Steel', 'final_steel', 'raw_blue_steel', 'black_steel'),
  ('red_steel', 'Red Steel', 'final_steel', 'raw_red_steel', 'black_steel');



-- 2) Now that every alloy ID exists, insert all `ingredients` rows.

-- Ingredients for simple alloys
INSERT INTO ingredients (alloy_id, ingredient_id, min_pct, max_pct) VALUES
  ('bismuth_bronze', 'copper', 50, 65),
  ('bismuth_bronze', 'zinc', 20, 30),
  ('bismuth_bronze', 'bismuth', 10, 20),
  ('black_bronze', 'copper', 50, 70),
  ('black_bronze', 'silver', 10, 25),
  ('black_bronze', 'gold', 10, 25),
  ('brass', 'copper', 88, 92),
  ('brass', 'zinc', 8, 12),
  ('rose_gold', 'gold', 70, 85),
  ('rose_gold', 'copper', 15, 30),
  ('sterling_silver', 'silver', 60, 80),
  ('sterling_silver', 'copper', 20, 40);

-- Ingredients for processed steel
INSERT INTO ingredients (alloy_id, ingredient_id, min_pct, max_pct) VALUES
  ('steel', 'pig_iron', 100, 100);

-- Ingredients for raw steels
INSERT INTO ingredients (alloy_id, ingredient_id, min_pct, max_pct) VALUES
  ('raw_black_steel', 'steel', 50, 70),
  ('raw_black_steel', 'nickel', 15, 25),
  ('raw_black_steel', 'black_bronze', 15, 25),
  ('raw_blue_steel', 'black_steel', 50, 55),
  ('raw_blue_steel', 'steel', 20, 25),
  ('raw_blue_steel', 'bismuth_bronze', 10, 15),
  ('raw_blue_steel', 'sterling_silver', 10, 15),
  ('raw_red_steel', 'black_steel', 50, 55),
  ('raw_red_steel', 'steel', 20, 25),
  ('raw_red_steel', 'brass', 10, 15),
  ('raw_red_steel', 'rose_gold', 10, 15);