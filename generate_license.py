import mysql.connector
import random
import string
from datetime import datetime, timedelta
import json
import os

# ==============================
#   CORES TERMINAL
# ==============================
class C:
    R = "\033[31m"  # vermelho
    G = "\033[32m"  # verde
    Y = "\033[33m"  # amarelo
    B = "\033[34m"  # azul
    P = "\033[35m"
    C = "\033[36m"
    W = "\033[37m"
    N = "\033[0m"   # reset


# ==============================
#   CONFIG
# ==============================
def load_config():
    with open("assets/config.json", "r") as f:
        data = json.load(f)
        return data["mysql"]

MYSQL = load_config()

# ==============================
#   DB
# ==============================
def db():
    return mysql.connector.connect(
        host=MYSQL["db_host"],
        user=MYSQL["db_user"],
        password=MYSQL["db_pass"],
        database=MYSQL["db_name"]
    )

# ==============================
#   LICENSE RANDOM
# ==============================
def generate_license():
    chars = string.ascii_letters + string.digits
    return ''.join(random.choice(chars) for _ in range(32))


# ==============================
#   CLEAR SCREEN
# ==============================
def cls():
    os.system("clear" if os.name != "nt" else "cls")


# ==============================
#   CRIAR LICENSE
# ==============================
def create_license():
    cls()
    print(f"{C.B}[CREATE LICENSE]{C.N}\n")

    try:
        days = int(input(f"{C.Y}Dias de validade: {C.N}"))
    except:
        print(f"{C.R}Valor inválido!{C.N}")
        return

    key = generate_license()
    expiry = (datetime.now() + timedelta(days=days)).strftime("%Y-%m-%d")

    conn = db()
    cur = conn.cursor()
    cur.execute("""
        INSERT INTO licenses (license_key, expiry_date, is_banned)
        VALUES (%s, %s, 0)
    """, (key, expiry))
    conn.commit()
    conn.close()

    print(f"\n{C.G}✔ License criada com sucesso!{C.N}")
    print(f"{C.C}License: {C.W}{key}")
    print(f"{C.C}Expira em: {C.W}{expiry}")
    print(f"{C.C}Banida: {C.W}0\n")


# ==============================
#   LIST LICENSES
# ==============================
def show_licenses():
    cls()
    print(f"{C.B}[LICENSE MANAGER]{C.N}\n")

    conn = db()
    cur = conn.cursor()
    cur.execute("SELECT id, license_key, expiry_date, is_banned FROM licenses")
    rows = cur.fetchall()
    conn.close()

    if not rows:
        print(f"{C.R}Nenhuma license encontrada.\n{C.N}")
        return

    print(f"{C.P}─────────────────────────────────────────────────────────────{C.N}")
    print(f"{C.W}ID   | STATUS    | EXPIRA       | LICENSE KEY{C.N}")
    print(f"{C.P}─────────────────────────────────────────────────────────────{C.N}")

    for row in rows:
        id_, key, expiry, banned = row

        status = f"{C.G}OK{C.N}" if not banned else f"{C.R}BANIDA{C.N}"

        print(f"{id_:<4} | {status:<9} | {expiry:<12} | {key}")

    print(f"{C.P}─────────────────────────────────────────────────────────────{C.N}\n")


# ==============================
#   BANIR LICENSE
# ==============================
def ban_license():
    cls()
    print(f"{C.B}[BAN LICENSE]{C.N}\n")
    key = input("License: ")

    conn = db()
    cur = conn.cursor()
    cur.execute("UPDATE licenses SET is_banned=1 WHERE license_key=%s", (key,))
    conn.commit()
    conn.close()

    print(f"\n{C.Y}✔ License banida!{C.N}\n")


# ==============================
#   DESBANIR LICENSE
# ==============================
def unban_license():
    cls()
    print(f"{C.B}[UNBAN LICENSE]{C.N}\n")
    key = input("License: ")

    conn = db()
    cur = conn.cursor()
    cur.execute("UPDATE licenses SET is_banned=0 WHERE license_key=%s", (key,))
    conn.commit()
    conn.close()

    print(f"\n{C.G}✔ License desbanida!{C.N}\n")


# ==============================
#   ADICIONAR DIAS
# ==============================
def add_days():
    cls()
    print(f"{C.B}[EXTENDER LICENÇA]{C.N}\n")

    key = input("License: ")
    days = int(input("Dias adicionais: "))

    conn = db()
    cur = conn.cursor()
    cur.execute("SELECT expiry_date FROM licenses WHERE license_key=%s", (key,))
    row = cur.fetchone()

    if not row:
        print(f"\n{C.R}❌ License não encontrada!{C.N}")
        return

    old = datetime.strptime(str(row[0]), "%Y-%m-%d")
    newdate = old + timedelta(days=days)
    new_str = newdate.strftime("%Y-%m-%d")

    cur.execute("UPDATE licenses SET expiry_date=%s WHERE license_key=%s", (new_str, key))
    conn.commit()
    conn.close()

    print(f"\n{C.G}✔ Dias adicionados!{C.N}")
    print(f"{C.W}Nova expiração: {new_str}\n")


# ==============================
#   DELETAR LICENSE
# ==============================
def delete_license():
    cls()
    print(f"{C.B}[DELETE LICENSE]{C.N}\n")
    key = input("License: ")

    conn = db()
    cur = conn.cursor()
    cur.execute("DELETE FROM licenses WHERE license_key=%s", (key,))
    conn.commit()
    conn.close()

    print(f"\n{C.R}✔ License deletada!{C.N}\n")


# ==============================
#   MENU
# ==============================
def main():
    while True:
        print(f"""
{C.C}================= LICENSE CNC ================={C.N}

{C.W}[1]{C.N} Criar nova license
{C.W}[2]{C.N} Listar todas licenses
{C.W}[3]{C.N} Banir uma license
{C.W}[4]{C.N} Desbanir uma license
{C.W}[5]{C.N} Adicionar dias à license
{C.W}[6]{C.N} Deletar license
{C.W}[0]{C.N} Sair
""")

        opt = input(f"{C.Y}Escolha: {C.N}")

        if opt == "1":
            create_license()
        elif opt == "2":
            show_licenses()
        elif opt == "3":
            ban_license()
        elif opt == "4":
            unban_license()
        elif opt == "5":
            add_days()
        elif opt == "6":
            delete_license()
        elif opt == "0":
            cls()
            break
        else:
            print(f"{C.R}\nOpção inválida!\n{C.N}")


if __name__ == "__main__":
    cls()
    main()
