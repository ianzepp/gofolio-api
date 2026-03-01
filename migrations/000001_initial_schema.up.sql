-- Initial schema migrated from Ghostfolio (Prisma-managed PostgreSQL)
-- Source: pg_dump of ghostfolio-db, Postgres 15

-- Enum types

CREATE TYPE "AccessPermission" AS ENUM (
    'READ',
    'READ_RESTRICTED'
);

CREATE TYPE "AssetClass" AS ENUM (
    'ALTERNATIVE_INVESTMENT',
    'COMMODITY',
    'EQUITY',
    'FIXED_INCOME',
    'LIQUIDITY',
    'REAL_ESTATE'
);

CREATE TYPE "AssetSubClass" AS ENUM (
    'BOND',
    'CASH',
    'COLLECTIBLE',
    'COMMODITY',
    'CRYPTOCURRENCY',
    'ETF',
    'MUTUALFUND',
    'PRECIOUS_METAL',
    'PRIVATE_EQUITY',
    'STOCK'
);

CREATE TYPE "DataSource" AS ENUM (
    'ALPHA_VANTAGE',
    'COINGECKO',
    'EOD_HISTORICAL_DATA',
    'FINANCIAL_MODELING_PREP',
    'GHOSTFOLIO',
    'GOOGLE_SHEETS',
    'MANUAL',
    'RAPID_API',
    'YAHOO'
);

CREATE TYPE "MarketDataState" AS ENUM (
    'CLOSE',
    'INTRADAY'
);

CREATE TYPE "Provider" AS ENUM (
    'ANONYMOUS',
    'GOOGLE',
    'INTERNET_IDENTITY',
    'OIDC'
);

CREATE TYPE "Role" AS ENUM (
    'ADMIN',
    'DEMO',
    'INACTIVE',
    'USER'
);

CREATE TYPE "Type" AS ENUM (
    'BUY',
    'DIVIDEND',
    'FEE',
    'INTEREST',
    'LIABILITY',
    'SELL'
);

CREATE TYPE "ViewMode" AS ENUM (
    'DEFAULT',
    'ZEN'
);

-- Tables

CREATE TABLE "User" (
    "accessToken" text,
    "authChallenge" text,
    "createdAt" timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    id text NOT NULL,
    provider "Provider" DEFAULT 'ANONYMOUS' NOT NULL,
    role "Role" DEFAULT 'USER' NOT NULL,
    "thirdPartyId" text,
    "updatedAt" timestamp(3) without time zone NOT NULL,
    CONSTRAINT "User_pkey" PRIMARY KEY (id)
);

CREATE TABLE "Platform" (
    id text NOT NULL,
    name text,
    url text NOT NULL,
    CONSTRAINT "Platform_pkey" PRIMARY KEY (id)
);

CREATE TABLE "Property" (
    key text NOT NULL,
    value text NOT NULL,
    CONSTRAINT "Property_pkey" PRIMARY KEY (key)
);

CREATE TABLE "Access" (
    alias text,
    "createdAt" timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "granteeUserId" text,
    id text NOT NULL,
    permissions "AccessPermission"[] DEFAULT ARRAY['READ_RESTRICTED'::"AccessPermission"],
    settings jsonb DEFAULT '{}' NOT NULL,
    "updatedAt" timestamp(3) without time zone NOT NULL,
    "userId" text NOT NULL,
    CONSTRAINT "Access_pkey" PRIMARY KEY (id)
);

CREATE TABLE "Account" (
    balance double precision DEFAULT 0 NOT NULL,
    comment text,
    "createdAt" timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    currency text,
    id text NOT NULL,
    "isExcluded" boolean DEFAULT false NOT NULL,
    name text,
    "platformId" text,
    "updatedAt" timestamp(3) without time zone NOT NULL,
    "userId" text NOT NULL,
    CONSTRAINT "Account_pkey" PRIMARY KEY (id, "userId")
);

CREATE TABLE "AccountBalance" (
    "accountId" text NOT NULL,
    "createdAt" timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    date timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    id text NOT NULL,
    "updatedAt" timestamp(3) without time zone NOT NULL,
    "userId" text NOT NULL,
    value double precision NOT NULL,
    CONSTRAINT "AccountBalance_pkey" PRIMARY KEY (id)
);

CREATE TABLE "Analytics" (
    "activityCount" integer DEFAULT 0 NOT NULL,
    country text,
    "dataProviderGhostfolioDailyRequests" integer DEFAULT 0 NOT NULL,
    "lastRequestAt" timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updatedAt" timestamp(3) without time zone NOT NULL,
    "userId" text NOT NULL,
    CONSTRAINT "Analytics_pkey" PRIMARY KEY ("userId")
);

CREATE TABLE "ApiKey" (
    "createdAt" timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "hashedKey" text NOT NULL,
    id text NOT NULL,
    "updatedAt" timestamp(3) without time zone NOT NULL,
    "userId" text NOT NULL,
    CONSTRAINT "ApiKey_pkey" PRIMARY KEY (id)
);

CREATE TABLE "AssetProfileResolution" (
    "createdAt" timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    currency text NOT NULL,
    "dataSourceOrigin" "DataSource" NOT NULL,
    "dataSourceTarget" "DataSource" NOT NULL,
    id text NOT NULL,
    "requestCount" integer DEFAULT 1 NOT NULL,
    "symbolOrigin" text NOT NULL,
    "symbolTarget" text NOT NULL,
    "updatedAt" timestamp(3) without time zone NOT NULL,
    CONSTRAINT "AssetProfileResolution_pkey" PRIMARY KEY (id)
);

CREATE TABLE "AuthDevice" (
    "createdAt" timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "credentialId" bytea NOT NULL,
    "credentialPublicKey" bytea NOT NULL,
    counter integer NOT NULL,
    id text NOT NULL,
    "updatedAt" timestamp(3) without time zone NOT NULL,
    "userId" text NOT NULL,
    CONSTRAINT "AuthDevice_pkey" PRIMARY KEY (id)
);

CREATE TABLE "MarketData" (
    "createdAt" timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "dataSource" "DataSource" NOT NULL,
    date timestamp(3) without time zone NOT NULL,
    id text NOT NULL,
    "marketPrice" double precision NOT NULL,
    state "MarketDataState" DEFAULT 'CLOSE' NOT NULL,
    symbol text NOT NULL,
    CONSTRAINT "MarketData_pkey" PRIMARY KEY (id)
);

CREATE TABLE "Order" (
    "accountId" text,
    "accountUserId" text,
    comment text,
    "createdAt" timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    currency text,
    date timestamp(3) without time zone NOT NULL,
    fee double precision NOT NULL,
    id text NOT NULL,
    "isDraft" boolean DEFAULT false NOT NULL,
    quantity double precision NOT NULL,
    "symbolProfileId" text NOT NULL,
    type "Type" NOT NULL,
    "unitPrice" double precision NOT NULL,
    "updatedAt" timestamp(3) without time zone NOT NULL,
    "userId" text NOT NULL,
    CONSTRAINT "Order_pkey" PRIMARY KEY (id)
);

CREATE TABLE "Settings" (
    settings jsonb,
    "updatedAt" timestamp(3) without time zone NOT NULL,
    "userId" text NOT NULL,
    CONSTRAINT "Settings_pkey" PRIMARY KEY ("userId")
);

CREATE TABLE "Subscription" (
    "createdAt" timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "expiresAt" timestamp(3) without time zone NOT NULL,
    id text NOT NULL,
    price double precision,
    "updatedAt" timestamp(3) without time zone NOT NULL,
    "userId" text NOT NULL,
    CONSTRAINT "Subscription_pkey" PRIMARY KEY (id)
);

CREATE TABLE "SymbolProfile" (
    "assetClass" "AssetClass",
    "assetSubClass" "AssetSubClass",
    comment text,
    countries jsonb,
    "createdAt" timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    currency text NOT NULL,
    cusip text,
    "dataSource" "DataSource" NOT NULL,
    figi text,
    "figiComposite" text,
    "figiShareClass" text,
    holdings jsonb DEFAULT '[]',
    id text NOT NULL,
    "isActive" boolean DEFAULT true NOT NULL,
    isin text,
    name text,
    "updatedAt" timestamp(3) without time zone NOT NULL,
    "scraperConfiguration" jsonb,
    sectors jsonb,
    symbol text NOT NULL,
    "symbolMapping" jsonb,
    url text,
    "userId" text,
    CONSTRAINT "SymbolProfile_pkey" PRIMARY KEY (id)
);

CREATE TABLE "SymbolProfileOverrides" (
    "assetClass" "AssetClass",
    "assetSubClass" "AssetSubClass",
    countries jsonb DEFAULT '[]',
    holdings jsonb DEFAULT '[]',
    name text,
    sectors jsonb DEFAULT '[]',
    "symbolProfileId" text NOT NULL,
    "updatedAt" timestamp(3) without time zone NOT NULL,
    url text,
    CONSTRAINT "SymbolProfileOverrides_pkey" PRIMARY KEY ("symbolProfileId")
);

CREATE TABLE "Tag" (
    id text NOT NULL,
    name text NOT NULL,
    "userId" text,
    CONSTRAINT "Tag_pkey" PRIMARY KEY (id)
);

CREATE TABLE "_OrderToTag" (
    "A" text NOT NULL,
    "B" text NOT NULL,
    CONSTRAINT "_OrderToTag_AB_pkey" PRIMARY KEY ("A", "B")
);

CREATE TABLE "_UserWatchlist" (
    "A" text NOT NULL,
    "B" text NOT NULL,
    CONSTRAINT "_UserWatchlist_AB_pkey" PRIMARY KEY ("A", "B")
);

-- Indexes

CREATE INDEX "Access_alias_idx" ON "Access" (alias);
CREATE INDEX "Access_granteeUserId_idx" ON "Access" ("granteeUserId");
CREATE INDEX "Access_userId_idx" ON "Access" ("userId");

CREATE UNIQUE INDEX "AccountBalance_accountId_date_key" ON "AccountBalance" ("accountId", date);
CREATE INDEX "AccountBalance_accountId_idx" ON "AccountBalance" ("accountId");
CREATE INDEX "AccountBalance_date_idx" ON "AccountBalance" (date);

CREATE INDEX "Account_currency_idx" ON "Account" (currency);
CREATE INDEX "Account_id_idx" ON "Account" (id);
CREATE INDEX "Account_isExcluded_idx" ON "Account" ("isExcluded");
CREATE INDEX "Account_name_idx" ON "Account" (name);
CREATE INDEX "Account_userId_idx" ON "Account" ("userId");

CREATE INDEX "Analytics_lastRequestAt_idx" ON "Analytics" ("lastRequestAt");
CREATE INDEX "Analytics_updatedAt_idx" ON "Analytics" ("updatedAt");

CREATE UNIQUE INDEX "ApiKey_hashedKey_key" ON "ApiKey" ("hashedKey");
CREATE INDEX "ApiKey_userId_idx" ON "ApiKey" ("userId");

CREATE UNIQUE INDEX "AssetProfileResolution_dataSourceOrigin_symbolOrigin_key" ON "AssetProfileResolution" ("dataSourceOrigin", "symbolOrigin");

CREATE INDEX "AuthDevice_userId_idx" ON "AuthDevice" ("userId");

CREATE UNIQUE INDEX "MarketData_dataSource_date_symbol_key" ON "MarketData" ("dataSource", date, symbol);
CREATE INDEX "MarketData_dataSource_idx" ON "MarketData" ("dataSource");
CREATE INDEX "MarketData_dataSource_symbol_idx" ON "MarketData" ("dataSource", symbol);
CREATE INDEX "MarketData_date_idx" ON "MarketData" (date);
CREATE INDEX "MarketData_marketPrice_idx" ON "MarketData" ("marketPrice");
CREATE INDEX "MarketData_state_idx" ON "MarketData" (state);
CREATE INDEX "MarketData_symbol_idx" ON "MarketData" (symbol);

CREATE INDEX "Order_accountId_idx" ON "Order" ("accountId");
CREATE INDEX "Order_date_idx" ON "Order" (date);
CREATE INDEX "Order_isDraft_idx" ON "Order" ("isDraft");
CREATE INDEX "Order_userId_idx" ON "Order" ("userId");

CREATE INDEX "Platform_name_idx" ON "Platform" (name);
CREATE UNIQUE INDEX "Platform_url_key" ON "Platform" (url);

CREATE INDEX "Subscription_userId_idx" ON "Subscription" ("userId");

CREATE INDEX "SymbolProfile_assetClass_idx" ON "SymbolProfile" ("assetClass");
CREATE INDEX "SymbolProfile_currency_idx" ON "SymbolProfile" (currency);
CREATE INDEX "SymbolProfile_cusip_idx" ON "SymbolProfile" (cusip);
CREATE INDEX "SymbolProfile_dataSource_idx" ON "SymbolProfile" ("dataSource");
CREATE UNIQUE INDEX "SymbolProfile_dataSource_symbol_key" ON "SymbolProfile" ("dataSource", symbol);
CREATE INDEX "SymbolProfile_isActive_idx" ON "SymbolProfile" ("isActive");
CREATE INDEX "SymbolProfile_isin_idx" ON "SymbolProfile" (isin);
CREATE INDEX "SymbolProfile_name_idx" ON "SymbolProfile" (name);
CREATE INDEX "SymbolProfile_symbol_idx" ON "SymbolProfile" (symbol);

CREATE INDEX "Tag_name_idx" ON "Tag" (name);
CREATE UNIQUE INDEX "Tag_name_userId_key" ON "Tag" (name, "userId");

CREATE INDEX "User_accessToken_idx" ON "User" ("accessToken");
CREATE INDEX "User_createdAt_idx" ON "User" ("createdAt");
CREATE INDEX "User_provider_idx" ON "User" (provider);
CREATE INDEX "User_role_idx" ON "User" (role);
CREATE INDEX "User_thirdPartyId_idx" ON "User" ("thirdPartyId");

CREATE INDEX "_OrderToTag_B_index" ON "_OrderToTag" ("B");
CREATE INDEX "_UserWatchlist_B_index" ON "_UserWatchlist" ("B");

-- Foreign keys

ALTER TABLE "Access" ADD CONSTRAINT "Access_granteeUserId_fkey"
    FOREIGN KEY ("granteeUserId") REFERENCES "User"(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "Access" ADD CONSTRAINT "Access_userId_fkey"
    FOREIGN KEY ("userId") REFERENCES "User"(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "AccountBalance" ADD CONSTRAINT "AccountBalance_accountId_userId_fkey"
    FOREIGN KEY ("accountId", "userId") REFERENCES "Account"(id, "userId") ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "Account" ADD CONSTRAINT "Account_platformId_fkey"
    FOREIGN KEY ("platformId") REFERENCES "Platform"(id) ON UPDATE CASCADE ON DELETE SET NULL;

ALTER TABLE "Account" ADD CONSTRAINT "Account_userId_fkey"
    FOREIGN KEY ("userId") REFERENCES "User"(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "Analytics" ADD CONSTRAINT "Analytics_userId_fkey"
    FOREIGN KEY ("userId") REFERENCES "User"(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "ApiKey" ADD CONSTRAINT "ApiKey_userId_fkey"
    FOREIGN KEY ("userId") REFERENCES "User"(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "AuthDevice" ADD CONSTRAINT "AuthDevice_userId_fkey"
    FOREIGN KEY ("userId") REFERENCES "User"(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "Order" ADD CONSTRAINT "Order_accountId_accountUserId_fkey"
    FOREIGN KEY ("accountId", "accountUserId") REFERENCES "Account"(id, "userId") ON UPDATE CASCADE ON DELETE SET NULL;

ALTER TABLE "Order" ADD CONSTRAINT "Order_symbolProfileId_fkey"
    FOREIGN KEY ("symbolProfileId") REFERENCES "SymbolProfile"(id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE "Order" ADD CONSTRAINT "Order_userId_fkey"
    FOREIGN KEY ("userId") REFERENCES "User"(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "Settings" ADD CONSTRAINT "Settings_userId_fkey"
    FOREIGN KEY ("userId") REFERENCES "User"(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "Subscription" ADD CONSTRAINT "Subscription_userId_fkey"
    FOREIGN KEY ("userId") REFERENCES "User"(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "SymbolProfileOverrides" ADD CONSTRAINT "SymbolProfileOverrides_symbolProfileId_fkey"
    FOREIGN KEY ("symbolProfileId") REFERENCES "SymbolProfile"(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "SymbolProfile" ADD CONSTRAINT "SymbolProfile_userId_fkey"
    FOREIGN KEY ("userId") REFERENCES "User"(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "Tag" ADD CONSTRAINT "Tag_userId_fkey"
    FOREIGN KEY ("userId") REFERENCES "User"(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "_OrderToTag" ADD CONSTRAINT "_OrderToTag_A_fkey"
    FOREIGN KEY ("A") REFERENCES "Order"(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "_OrderToTag" ADD CONSTRAINT "_OrderToTag_B_fkey"
    FOREIGN KEY ("B") REFERENCES "Tag"(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "_UserWatchlist" ADD CONSTRAINT "_UserWatchlist_A_fkey"
    FOREIGN KEY ("A") REFERENCES "SymbolProfile"(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE "_UserWatchlist" ADD CONSTRAINT "_UserWatchlist_B_fkey"
    FOREIGN KEY ("B") REFERENCES "User"(id) ON UPDATE CASCADE ON DELETE CASCADE;
