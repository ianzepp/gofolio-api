-- Drop foreign keys first
ALTER TABLE "_UserWatchlist" DROP CONSTRAINT IF EXISTS "_UserWatchlist_B_fkey";
ALTER TABLE "_UserWatchlist" DROP CONSTRAINT IF EXISTS "_UserWatchlist_A_fkey";
ALTER TABLE "_OrderToTag" DROP CONSTRAINT IF EXISTS "_OrderToTag_B_fkey";
ALTER TABLE "_OrderToTag" DROP CONSTRAINT IF EXISTS "_OrderToTag_A_fkey";
ALTER TABLE "Tag" DROP CONSTRAINT IF EXISTS "Tag_userId_fkey";
ALTER TABLE "SymbolProfile" DROP CONSTRAINT IF EXISTS "SymbolProfile_userId_fkey";
ALTER TABLE "SymbolProfileOverrides" DROP CONSTRAINT IF EXISTS "SymbolProfileOverrides_symbolProfileId_fkey";
ALTER TABLE "Subscription" DROP CONSTRAINT IF EXISTS "Subscription_userId_fkey";
ALTER TABLE "Settings" DROP CONSTRAINT IF EXISTS "Settings_userId_fkey";
ALTER TABLE "Order" DROP CONSTRAINT IF EXISTS "Order_userId_fkey";
ALTER TABLE "Order" DROP CONSTRAINT IF EXISTS "Order_symbolProfileId_fkey";
ALTER TABLE "Order" DROP CONSTRAINT IF EXISTS "Order_accountId_accountUserId_fkey";
ALTER TABLE "AuthDevice" DROP CONSTRAINT IF EXISTS "AuthDevice_userId_fkey";
ALTER TABLE "ApiKey" DROP CONSTRAINT IF EXISTS "ApiKey_userId_fkey";
ALTER TABLE "Analytics" DROP CONSTRAINT IF EXISTS "Analytics_userId_fkey";
ALTER TABLE "Account" DROP CONSTRAINT IF EXISTS "Account_userId_fkey";
ALTER TABLE "Account" DROP CONSTRAINT IF EXISTS "Account_platformId_fkey";
ALTER TABLE "AccountBalance" DROP CONSTRAINT IF EXISTS "AccountBalance_accountId_userId_fkey";
ALTER TABLE "Access" DROP CONSTRAINT IF EXISTS "Access_userId_fkey";
ALTER TABLE "Access" DROP CONSTRAINT IF EXISTS "Access_granteeUserId_fkey";

-- Drop tables (reverse dependency order)
DROP TABLE IF EXISTS "_UserWatchlist";
DROP TABLE IF EXISTS "_OrderToTag";
DROP TABLE IF EXISTS "Tag";
DROP TABLE IF EXISTS "SymbolProfileOverrides";
DROP TABLE IF EXISTS "SymbolProfile";
DROP TABLE IF EXISTS "Subscription";
DROP TABLE IF EXISTS "Settings";
DROP TABLE IF EXISTS "Order";
DROP TABLE IF EXISTS "MarketData";
DROP TABLE IF EXISTS "AuthDevice";
DROP TABLE IF EXISTS "AssetProfileResolution";
DROP TABLE IF EXISTS "ApiKey";
DROP TABLE IF EXISTS "Analytics";
DROP TABLE IF EXISTS "AccountBalance";
DROP TABLE IF EXISTS "Account";
DROP TABLE IF EXISTS "Access";
DROP TABLE IF EXISTS "Property";
DROP TABLE IF EXISTS "Platform";
DROP TABLE IF EXISTS "User";

-- Drop enum types
DROP TYPE IF EXISTS "ViewMode";
DROP TYPE IF EXISTS "Type";
DROP TYPE IF EXISTS "Role";
DROP TYPE IF EXISTS "Provider";
DROP TYPE IF EXISTS "MarketDataState";
DROP TYPE IF EXISTS "DataSource";
DROP TYPE IF EXISTS "AssetSubClass";
DROP TYPE IF EXISTS "AssetClass";
DROP TYPE IF EXISTS "AccessPermission";
