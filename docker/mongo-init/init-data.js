//let primary = rs.status().members.find(m => m.stateStr === "PRIMARY").name;
//db = connect(primary + "/testdb");

//db = db.getSiblingDB('testdb');

use testdb;

db.users.insertMany([
  { name: "John Doe", email: "john@example.com", age: 30 },
  { name: "Jane Smith", email: "jane@example.com", age: 28 },
  { name: "Bob Johnson", email: "bob@example.com", age: 35 }
]);

print("Initial data inserted into testdb database.");
