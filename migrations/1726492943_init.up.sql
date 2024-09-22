create table if not exists "user"
(
    id            bigserial
        constraint user_pk
            primary key,
    login         varchar(100) not null,
    password_hash varchar(255) not null
);

create unique index if not exists user_login_uindex
    on "user" (login);

create table if not exists orders
(
    id         bigserial
        constraint orders_pk
            primary key,
    number     varchar(32)                               not null,
    status     varchar(20)                               not null,
    user_id    bigint                                    not null
        constraint orders_user_id_fk
            references "user"
            on delete cascade,
    created_at timestamp(3) with time zone default now() not null
);

create unique index if not exists orders_number_uindex
    on orders (number);

create table if not exists balance
(
    id           bigserial
        constraint id
            primary key,
    order_number varchar(32)                               not null,
    sum          numeric(31, 11)                           not null,
    type         smallint                                  not null,
    created_at   timestamp(3) with time zone default now() not null,
    user_id      bigint                                    not null
        constraint balance_user_id_fk
            references "user"
            on delete cascade
);

create unique index if not exists balance_order_number_type_uindex
    on balance (order_number, type)
    where (type = 0);

create index if not exists balance_user_id_index
    on balance (user_id);
