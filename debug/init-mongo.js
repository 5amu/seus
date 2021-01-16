db.createUser(
    {
        user  : "test",
        pwd   : "test",
        roles : [
            {
                role : "readWrite",
                db   : "seus-debug"
            }
        ]
    }
)