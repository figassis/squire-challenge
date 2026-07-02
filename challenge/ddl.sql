-- DROP SCHEMA public;

CREATE SCHEMA public AUTHORIZATION pg_database_owner;

COMMENT ON SCHEMA public IS 'standard public schema';

-- DROP TYPE public.entry_direction;

CREATE TYPE public.entry_direction AS ENUM (
	'credit',
	'debit');

-- DROP TYPE public.entry_layer;

CREATE TYPE public.entry_layer AS ENUM (
	'pending',
	'settled',
	'encumbrance');

-- DROP SEQUENCE public.account_account_id_seq;

CREATE SEQUENCE public.account_account_id_seq
	INCREMENT BY 1
	MINVALUE 1
	MAXVALUE 9223372036854775807
	START 1
	CACHE 1
	NO CYCLE;-- public.account definition

-- Drop table

-- DROP TABLE public.account;

CREATE TABLE public.account (
	account_id int8 GENERATED ALWAYS AS IDENTITY( INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START 1 CACHE 1 NO CYCLE) NOT NULL,
	account_name text NOT NULL,
	CONSTRAINT account_pkey PRIMARY KEY (account_id)
);


-- public.balance definition

-- Drop table

-- DROP TABLE public.balance;

CREATE TABLE public.balance (
	account_id int8 NOT NULL,
	pending_balance numeric(20, 4) DEFAULT 0 NOT NULL,
	settled_balance numeric(20, 4) DEFAULT 0 NOT NULL,
	encumbrance_balance numeric(20, 4) DEFAULT 0 NOT NULL,
	CONSTRAINT balance_pkey PRIMARY KEY (account_id),
	CONSTRAINT balance_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account(account_id)
);


-- public.transactions definition

-- Drop table

-- DROP TABLE public.transactions;

CREATE TABLE public.transactions (
	entry_id varchar(500) NOT NULL,
	transaction_id varchar(500) NOT NULL,
	account_id int8 NOT NULL,
	amount numeric(20, 4) NOT NULL,
	direction public.entry_direction NOT NULL,
	layer public.entry_layer NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT transactions_amount_check CHECK ((amount >= (0)::numeric)),
	CONSTRAINT transactions_pkey PRIMARY KEY (entry_id),
	CONSTRAINT transactions_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account(account_id)
);