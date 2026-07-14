yeah your .env needs a DB_URL variable and a JWT_SECRET variable for this to work

make the jwt secret with
openssl rand -base64 64
for Linux. I don't know about other systems tbh.