CREATE TABLE songs (
                             id BIGSERIAL PRIMARY KEY,
                             song_name VARCHAR(255) NOT NULL,
                             group_name VARCHAR(255) NOT NULL,
                             created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
                             updated_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
                             release VARCHAR(255),
                             text VARCHAR(255),
                             link VARCHAR(255)
);

CREATE UNIQUE INDEX unique_song_group ON songs (song_name, group_name);


