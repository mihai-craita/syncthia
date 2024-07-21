#!/bin/bash

echo "Waiting for MongoDB to start..."
sleep 20

echo "Initializing replica set..."
mongosh --host mongo1:27017 <<EOF
rs.initiate({
  _id: "rs0",
  members: [
    {_id: 0, host: "mongo1:27017"},
    {_id: 1, host: "mongo2:27018"},
    {_id: 2, host: "mongo3:27019"}
  ]
});

use testdb;

db.users.insertMany([
  { name: "John Doe", email: "john@example.com", age: 30 },
  { name: "Jane Smith", email: "jane@example.com", age: 28 },
  { name: "Bob Johnson", email: "bob@example.com", age: 35 }
]);
EOF

echo "Replica set initialized."
