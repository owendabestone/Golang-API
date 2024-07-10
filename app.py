from flask import Flask
from flask import render_template, redirect, url_for
from flask import request
import sqlalchemy as db
from sqlalchemy import create_engine
from sqlalchemy import text
import json

app = Flask(__name__)
# app.config["SERVER_NAME"] = "owenzhangvm:8001"
# ROOT = "C:\\Users\\e150907\\apps\\Binformed_house_keeping\\"
engine = None
@app.route('/fetch', methods=['POST'])
def fetch():
    cleaned = request.data.decode('utf-8').replace('\t','').replace('\n','')
    body = json.loads(cleaned)
    # json_str = request.data.decode('utf-8')
    # json_objs = json_str.split('\n')
    # for obj in json_objs:
    #     print(json.loads(obj))


    pricing_group_raw = body['pricing_group']
    branch_raw = body['branch']
    branch = ''.join(letter for letter in branch_raw if letter.isalnum())
    pricing_group = ''.join(letter for letter in pricing_group_raw if letter.isalnum())

    with engine.connect() as connection:
        r = connection.execute(text(f"SELECT cust_name, cust_code , seq_num FROM PriceGroup WHERE group_id = '{pricing_group}' AND branch='{branch}'"))
        rows = r.fetchall()
        result = []
        for index in range(len(rows)):
            result.append({'name':rows[index][0],'id':rows[index][1],'shipto':rows[index][2]})
    # Convert the list of dictionaries to JSON and print it
    json_result = json.dumps({'response':result})
    return json_result

if __name__ == "__main__":

    user = 'Groupie01'
    password = 'Groupie2024!'
    host = 'bdsql2016Prod'
    port = 1433
    database = 'PriceGroup'
    
    # PYTHON FUNCTION TO CONNECT TO THE MYSQL DATABASE AND
    # RETURN THE SQLACHEMY ENGINE OBJECT
    def get_connection():
        return create_engine(
            url="mssql+pyodbc://{0}:{1}@{2}:{3}/{4}?driver=SQL+Server".format(
                user, password, host, port, database
            )
        )
    engine = get_connection()

    app.run(debug= True)