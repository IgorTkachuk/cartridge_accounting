package doc

import (
	"context"
	"errors"
	"fmt"
	"github.com/IgorTkachuk/cartridge_accounting/internal/domain/doc"
	"github.com/IgorTkachuk/cartridge_accounting/pkg/client/postgresql"
	"github.com/IgorTkachuk/cartridge_accounting/pkg/logging"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"time"
)

var _ doc.Repository = &repository{}

type repository struct {
	client postgresql.Client
	logger *logging.Logger
}

func (r repository) FindAll(ctx context.Context) ([]doc.Doc, error) {
	q := `
		SELECT 
			id, doc_type_id, doc_date, employee_id, doc_owner_id, 
				decommissioning_cause_id, ou_id, sd_claim_number,
					regenerate_type_id, created_at, updated_at
		FROM
			doc
	`
	rows, err := r.client.Query(ctx, q)
	if err != nil {
		return nil, err
	}

	docs := make([]doc.Doc, 0)
	for rows.Next() {
		var doc doc.Doc
		err := rows.Scan(
			&doc.ID, &doc.DocTypeId, &doc.DocDate, &doc.EmployeeId, &doc.DocOwnerId,
			&doc.DecommissioningCauseId, &doc.OuId, &doc.SdClaimNumber,
			&doc.RegenerateTypeId, &doc.CreatedAt, &doc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

func (r repository) FindById(ctx context.Context, id int) (d doc.Doc, err error) {
	q := `
		SELECT 
			id, doc_type_id, doc_date, employee_id, doc_owner_id, 
				decommissioning_cause_id, ou_id, sd_claim_number,
					regenerate_type_id, created_at, updated_at
		FROM
			doc
		WHERE
			id=$1
	`
	err = r.client.QueryRow(ctx, q, id).Scan(
		&d.ID, &d.DocTypeId, &d.DocDate, &d.EmployeeId, &d.DocOwnerId,
		&d.DecommissioningCauseId, &d.OuId, &d.SdClaimNumber,
		&d.RegenerateTypeId, &d.CreatedAt, &d.UpdatedAt,
	)

	if err != nil {
		return doc.Doc{}, err
	}

	return
}

func (r repository) Create(ctx context.Context, dto doc.CreateDocDTO) (docId int, err error) {
	qDoc := `
		INSERT INTO doc (
			doc_type_id, doc_date, employee_id, doc_owner_id, 
			decommissioning_cause_id, ou_id, sd_claim_number,
			regenerate_type_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	tx, err := r.client.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, qDoc,
		dto.DocTypeId, dto.DocDate, dto.EmployeeId, dto.DocOwnerId,
		dto.DecommissioningCauseId, dto.OuId, dto.SdClaimNumber,
		dto.RegenerateTypeId,
	).Scan(&docId)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, pgErr) {
			pgErr = err.(*pgconn.PgError)
			newErr := fmt.Errorf(fmt.Sprintf("%s", pgErr.Message))
			return -1, newErr
		}
		return -1, err
	}

	qCtr := `
		INSERT INTO ctr(
			model_id, sn
		)
		VALUES ($1, $2)
		RETURNING id
	`
	qDocCtr := `
		INSERT INTO doc_ctr (
			doc_id, ctr_id
		)
		VALUES ($1, $2)
	`
	qGetNewCtrStatus := `
		SELECT ctr_status_type_to
		FROM doc_type
		WHERE id = $1
	`
	qSetCtrStatus := `
		INSERT INTO ctr_status (
			ctr_id, status_id, doc_id
		)
		VALUES ($1, $2, $3)
	`
	for _, ctr := range dto.Ctrs {
		var ctrId int
		err := tx.QueryRow(ctx, qCtr, ctr.ModelId, ctr.SerialNumber).Scan(&ctrId)
		if err != nil {
			return 0, err
		}
		_, err = tx.Exec(ctx, qDocCtr, docId, ctrId)
		if err != nil {
			return 0, err
		}

		var ctrNewStatus int
		err = tx.QueryRow(ctx, qGetNewCtrStatus, dto.DocTypeId).Scan(&ctrNewStatus)
		if err != nil {
			return 0, err
		}

		_, err = tx.Exec(ctx, qSetCtrStatus, ctrId, ctrNewStatus, docId)
		if err != nil {
			return 0, err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return 0, err
	}

	return
}

func (r repository) Update(ctx context.Context, dto doc.UpdateDocDTO) error {
	q := `
		UPDATE doc
		SET 
			doc_type_id=$1, doc_date=$2, employee_id=$3, doc_owner_id=$4, 
			decommissioning_cause_id=$5, ou_id=$6, sd_claim_number=$7,
			regenerate_type_id=$8, updated_at=$9
		WHERE
			id=$10
			
	`
	_, err := r.client.Exec(ctx, q,
		dto.DocTypeId, dto.DocDate, dto.EmployeeId, dto.DocOwnerId,
		dto.DecommissioningCauseId, dto.OuId, dto.SdClaimNumber,
		dto.RegenerateTypeId, time.Now().UTC(), dto.ID,
	)

	if err != nil {
		return err
	}

	return nil
}

func (r repository) Delete(ctx context.Context, id int) error {
	q := `
		DELETE 
		FROM doc
		WHERE id=$1
	`
	qGetCtr := `
		SELECT ctr_id
		FROM doc_ctr
		WHERE doc_id = $1
	`
	qCtrDocs := `
		SELECT COUNT(*)
		FROM doc_ctr
		WHERE ctr_id = $1
	`
	qCtrDel := `
		DELETE FROM ctr WHERE id = $1
	`

	ctrIdRows, err := r.client.Query(ctx, qGetCtr, id)
	if err != nil {
		return err
	}

	_, err = r.client.Exec(ctx, q, id)
	if err != nil {
		return err
	}

	for ctrIdRows.Next() {
		var ctrId int
		err := ctrIdRows.Scan(&ctrId)
		if err != nil {
			return err
		}
		var ctrDocCount int
		err = r.client.QueryRow(ctx, qCtrDocs, ctrId).Scan(&ctrDocCount)
		if err != nil {
			return err
		}
		if ctrDocCount == 0 {
			_, err = r.client.Exec(ctx, qCtrDel, ctrId)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewRepository(client postgresql.Client, logger *logging.Logger) *repository {
	return &repository{client: client, logger: logger}
}
