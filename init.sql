-- Written by Gemini CLI
-- This file is licensed under the MIT License.
-- See the LICENSE file for details.

CREATE TABLE IF NOT EXISTS todos (
    id SERIAL PRIMARY KEY,
    task TEXT NOT NULL,
    completed BOOLEAN DEFAULT FALSE
);
