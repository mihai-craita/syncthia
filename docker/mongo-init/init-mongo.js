db.createUser({
    user: 'admin',
    pwd: 'password',
    roles: [
        {
            role: 'readWrite',
            db: 'testdb'
        }
    ]
});

db = new Mongo().getDB('testdb');

db.users.insert([
    { name: "Charlie", email: "charlie@example.com" },
    { name: "Dave", email: "dave@example.com" }
]);

